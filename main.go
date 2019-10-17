package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/Hsn723/nginx-i2c/i2c"
	"github.com/oschwald/maxminddb-golang"
	"io"
	"io/ioutil"
	"log"
	"math/bits"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
)

const (
	geoliteURL = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
)

var (
	// AFRINIC, APNIC, ARIN. LACNIC, RIPE
	rirURLs = []string{
		"https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest",
		"https://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest",
		"https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest",
		"https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest",
		"https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest",
	}
)

func getMMDBSubnets(mmdb *maxminddb.Reader, entries map[string]string) error {
	networks := mmdb.Networks()
	var r i2c.Record
	for networks.Next() {
		subnet, err := networks.Network(&r)
		if err != nil {
			return err
		}
		if r.IsAnonymousProxy || r.IsSatelliteProvider {
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
	// TODO: better IP address sorting
	sort.Strings(subnets)
	return
}

func ipCountToSubnetMask(count uint32) (mask int) {
	bits := bits.Len32(count) - 1
	mask = 32 - bits
	return
}

func isIgnoredLine(line []string) bool {
	if _, err := strconv.ParseFloat(line[0], 64); err == nil {
		return true
	}
	if line[len(line)-1] == "summary" {
		return true
	}
	if line[2] == "asn" {
		return true
	}
	// TODO: filter out countries in ignore list
	return false
}

func appendRIRSubnets(mmdb *maxminddb.Reader, csvReader *csv.Reader, entries map[string]string) error {
	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if isIgnoredLine(line) {
			continue
		}
		ip := net.ParseIP(line[3])
		var r i2c.Record
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

func appendAllRIRSubnets(mmdb *maxminddb.Reader, entries map[string]string, dir string) error {
	for _, rir := range rirURLs {
		rirFile, e := i2c.GetRIRFile(rir, dir)
		if e != nil {
			return e
		}
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
		if e := appendRIRSubnets(mmdb, reader, entries); e != nil {
			return e
		}
	}
	return nil
}

func writeI2C(entries map[string]string, filename string, tmpDir string) (err error) {
	tmpFilePath := path.Join(tmpDir, filename)

	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	outFilePath := path.Join(cwd, filename)
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		return
	}
	subnets := getSortedSubnets(entries)
	for _, subnet := range subnets {
		_, e := tmpFile.WriteString(fmt.Sprintf("%s %s;\n", subnet, entries[subnet]))
		if e != nil {
			err = e
			return
		}
	}
	err = os.Rename(tmpFilePath, outFilePath)
	return
}

func main() {
	workDir, err := ioutil.TempDir("", "i2c")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)
	log.Printf("Created temporary directory %s", workDir)
	mmdbFilename, err := i2c.GetMMDBFile(geoliteURL, workDir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mmdb, err := maxminddb.Open(mmdbFilename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	entries := map[string]string{}

	if err := getMMDBSubnets(mmdb, entries); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	if err := appendAllRIRSubnets(mmdb, entries, workDir); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if err := writeI2C(entries, "ip2country.conf", workDir); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Println("Wrote .conf file")
}
