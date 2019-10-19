package i2c

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/oschwald/maxminddb-golang"
	"io"
	"math/bits"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

func ipCountToSubnetMask(count uint32) (mask int) {
	bits := bits.Len32(count) - 1
	mask = 32 - bits
	return
}

func isIgnoredLine(line []string, isIPv4Only bool) bool {
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
	// TODO: filter out countries in ignore list
	return false
}

// exclude IPv4 mapped IPv6 addresses
func isIPv4(ip net.IP) bool {
	return ip.To4() != nil && strings.Count(ip.String(), ":") < 2
}

// GetMMDBSubnets extracts subnets from the given MaxMind database
func GetMMDBSubnets(mmdb *maxminddb.Reader, entries map[string]string, isIPv4Only bool) error {
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
		if country == "" {
			continue
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

func appendRIRSubnets(mmdb *maxminddb.Reader, csvReader *csv.Reader, entries map[string]string, isIPv4Only bool) error {
	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if isIgnoredLine(line, isIPv4Only) {
			continue
		}
		ip := net.ParseIP(line[3])
		var r Record
		_, found, err := mmdb.LookupNetwork(ip, &r)
		if err != nil {
			return err
		}
		if !found {
			count, err := strconv.ParseUint(line[4], 10, 32)
			if err != nil {
				return err
			}
			maskPart := ipCountToSubnetMask(uint32(count))
			newSubnet := fmt.Sprintf("%s/%v", ip, maskPart)
			country := line[1]
			entries[newSubnet] = country
			continue
		}
		// TODO, handle if found and not matching?
	}
	return nil
}

// AppendAllRIRSubnets uses RIR entries to add missing records to the MaxMind database
func AppendAllRIRSubnets(mmdb *maxminddb.Reader, entries map[string]string, rirFiles []string, isIPv4Only bool) error {
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
		if e := appendRIRSubnets(mmdb, reader, entries, isIPv4Only); e != nil {
			return e
		}
	}
	return nil
}

// GetDBReader opens the mmdb file
func GetDBReader(filename string) (*maxminddb.Reader, error) {
	return maxminddb.Open(filename)
}
