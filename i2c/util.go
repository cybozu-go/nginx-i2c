package i2c

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/oschwald/maxminddb-golang"
	"go4.org/netipx"
)

var defaultIgnoredCountries = map[string]struct{}{
	"":   {},
	"ZZ": {},
}

// CountrySliceToMap converts a slice to a map for faster lookups
func CountrySliceToMap(cc []string) (countries map[string]struct{}) {
	countries = make(map[string]struct{})
	for _, c := range cc {
		c = strings.ToUpper(c)
		if _, ok := countries[c]; !ok {
			countries[c] = struct{}{}
		}
	}
	return
}

func containsCountry(countries map[string]struct{}, country string) (contained bool) {
	country = strings.ToUpper(country)
	_, contained = countries[country]
	return
}

func isIgnoredCountry(country string, includeCountries, excludeCountries map[string]struct{}) bool {
	if containsCountry(defaultIgnoredCountries, country) {
		return true
	}
	if containsCountry(excludeCountries, country) {
		return true
	}
	return len(includeCountries) > 0 && !containsCountry(includeCountries, country)
}

func isIgnoredLine(line []string, isIPv4Only bool, includeCountries, excludeCountries map[string]struct{}) bool {
	if _, err := strconv.ParseFloat(line[0], 64); err == nil {
		return true
	}
	if line[len(line)-1] == "summary" {
		return true
	}
	if isIPv4Only && line[2] != "ipv4" {
		return true
	}
	if line[2] == "asn" {
		return true
	}
	return isIgnoredCountry(line[1], includeCountries, excludeCountries)
}

// exclude IPv4 mapped IPv6 addresses
func isIPv4(ip net.IP) bool {
	return ip.To4() != nil && strings.Count(ip.String(), ":") < 2
}

// GetMMDBSubnets extracts subnets from the given MaxMind database
func GetMMDBSubnets(mmdb *maxminddb.Reader, entries map[string]string, isIPv4Only, lowercase bool, includeCountries, excludeCountries map[string]struct{}) error {
	networks := mmdb.Networks()
	var r Record
	for networks.Next() {
		subnet, err := networks.Network(&r)
		if err != nil {
			return err
		}
		if r.IsAnonymousProxy || r.IsSatelliteProvider {
			continue
		}
		if isIPv4Only && !isIPv4(subnet.IP) {
			continue
		}
		country := r.Country.IsoCode
		if country == "" {
			country = r.RegisteredCountry.IsoCode
		}
		if isIgnoredCountry(country, includeCountries, excludeCountries) {
			continue
		}
		if lowercase {
			country = strings.ToLower(country)
		}
		entries[subnet.String()] = country
	}
	if networks.Err() != nil {
		return networks.Err()
	}
	return nil
}

func getSortedSubnets(entries map[string]string) (subnets []string) {
	subnets = make([]string, 0, len(entries))
	for s := range entries {
		subnets = append(subnets, s)
	}
	sort.Slice(subnets, func(i, j int) bool {
		ip1, _, _ := net.ParseCIDR(subnets[i])
		ip2, _, _ := net.ParseCIDR(subnets[j])
		return bytes.Compare(ip1, ip2) < 0
	})
	return
}

func getSubnetsFromIPCount(startIP string, count uint32) ([]netip.Prefix, error) {
	start, err := netip.ParseAddr(startIP)
	if err != nil {
		return nil, err
	}
	endInt := new(big.Int).SetBytes(start.AsSlice())
	endInt = endInt.Add(endInt, big.NewInt(int64(count) - 1))
	end, ok := netip.AddrFromSlice(endInt.Bytes())
	if !ok {
		return nil, fmt.Errorf("invalid IP %s", endInt)
	}
	ipRange := netipx.IPRangeFrom(start, end)
	return ipRange.Prefixes(), nil
}

func appendRIRSubnets(mmdb *maxminddb.Reader, csvReader *csv.Reader, entries map[string]string, isIPv4Only, lowercase bool, includeCountries, excludeCountries map[string]struct{}) error {
	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if isIgnoredLine(line, isIPv4Only, includeCountries, excludeCountries) {
			continue
		}
		ip := net.ParseIP(line[3])
		var r Record
		_, found, err := mmdb.LookupNetwork(ip, &r)
		if err != nil {
			return err
		}
		if !found {
			country := line[1]
			if lowercase {
				country = strings.ToLower(country)
			}
			maskPart, err := strconv.ParseUint(line[4], 10, 32)
			if err != nil {
				return err
			}
			mask := uint32(maskPart)
			if !isIPv4(ip) {
				subnet := fmt.Sprintf("%s/%v", ip, mask)
				entries[subnet] = country
				continue
			}
			subnets, err := getSubnetsFromIPCount(line[3], mask)
			if err != nil {
				return err
			}
			for _, subnet := range subnets {
				startIP := net.ParseIP(subnet.Addr().String())
				if _, found, _ := mmdb.LookupNetwork(startIP, &Record{}); found {
					continue
				}
				entries[subnet.String()] = country
			}
			continue
		}
	}
	return nil
}

// AppendAllRIRSubnets uses RIR entries to add missing records to the MaxMind database
func AppendAllRIRSubnets(mmdb *maxminddb.Reader, entries map[string]string, rirFiles []string, isIPv4Only, lowercase bool, includeCountries, excludeCountries map[string]struct{}) error {
	for _, rirFile := range rirFiles {
		csvFile, e := os.Open(rirFile)
		if e != nil {
			return e
		}
		defer csvFile.Close()
		reader := csv.NewReader(bufio.NewReader(csvFile))
		reader.Comma = '|'
		reader.Comment = '#'
		// some delegated dbs are not uniform
		reader.FieldsPerRecord = -1
		if e := appendRIRSubnets(mmdb, reader, entries, isIPv4Only, lowercase, includeCountries, excludeCountries); e != nil {
			return e
		}
	}
	return nil
}

// GetDBReader opens the mmdb file
func GetDBReader(filename string) (*maxminddb.Reader, error) {
	return maxminddb.Open(filename)
}
