package main

import (
	"errors"
	"github.com/mholt/archiver"
	"github.com/oschwald/maxminddb-golang"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const (
	geoliteURL = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
	afrinicURL = "https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest"
	apnicURL = "https://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest"
	arinURL = "https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"
	lacnicURL = "https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest"
	ripeURL = "https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest"
)

type record struct {
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	RegisteredCountry struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"registered_country"`
	Traits struct {
		IsAnonymousProxy    bool `maxminddb:"is_anonymous_proxy"`
		IsSatelliteProvider bool `maxminddb:"is_satellite_provider"`
	} `maxminddb:"traits"`
}

func downloadArchive(url string, baseDir string) (filename string, err error) {
	r, err := http.Get(url)
	if err != nil {
		return
	}
	filePath := path.Join(baseDir, path.Base(url))
	out, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer out.Close()
	_, err = io.Copy(out, r.Body)
	if err != nil {
		return
	}
	filename = filePath
	log.Printf("Downloaded %s", filename)
	return
}

func extractMaxMindDB(filePath string, baseDir string) (filename string, err error) {
	var mmdbFilename string
	_ = archiver.Walk(filePath, func(f archiver.File) error {
		if filepath.Ext(f.Name()) == ".mmdb" {
			mmdbFilename = path.Join(baseDir, f.Name())
			log.Printf("Found %s", mmdbFilename)
			// Extract right away
			out, err := os.Create(mmdbFilename)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, f.ReadCloser)
			if err != nil {
				return err
			}
			return archiver.ErrStopWalk
		}
		return nil
	})
	if mmdbFilename == "" {
		err = errors.New("Could not find .mmdb file in archive")
		return
	}
	filename = mmdbFilename
	log.Printf("Extracted %s", mmdbFilename)
	return
}

func main() {
	dir, err := ioutil.TempDir("", "i2c")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	log.Printf("Created temporary directory %s", dir)
	glTarPath, err := downloadArchive(geoliteURL, dir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mmdbFilename, err := extractMaxMindDB(glTarPath, dir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mmdb, err := maxminddb.Open(mmdbFilename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	var r record

	networks := mmdb.Networks()
	for networks.Next() {
		subnet, err := networks.Network(&r)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		log.Printf("%s: %s/%s, %t %t", subnet.String(), r.Country.IsoCode, r.RegisteredCountry.IsoCode, r.Traits.IsAnonymousProxy, r.Traits.IsSatelliteProvider)
	}
	if networks.Err() != nil {
		log.Fatal(networks.Err())
	}
}
