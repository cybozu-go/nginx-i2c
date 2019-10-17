package main

import (
	"fmt"
	"github.com/Hsn723/nginx-i2c/i2c"
	"github.com/oschwald/maxminddb-golang"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
)

const (
	geoliteURL = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
	afrinicURL = "https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest"
	apnicURL   = "https://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest"
	arinURL    = "https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"
	lacnicURL  = "https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest"
	ripeURL    = "https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest"
)

var workDir string

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
	sort.Strings(subnets)
	return
}

func writeI2C(entries map[string]string, filename string) (err error) {
	tmpFilePath := path.Join(workDir, filename)

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
	err = getMMDBSubnets(mmdb, entries)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	// TODO: parse RIRs
	err = writeI2C(entries, "ip2country.conf")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Println("Wrote .conf file")
}
