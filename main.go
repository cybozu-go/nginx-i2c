package main

import (
	"errors"
	"github.com/oschwald/maxminddb-golang"
	"github.com/mholt/archiver"
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
)

func DownloadArchive(url string, baseDir string) (filename string, err error) {
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

func ExtractMaxMindDB(filePath string, baseDir string) (filename string, err error) {
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
	glTarPath, err := DownloadArchive(geoliteURL, dir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mmdbFilename, err := ExtractMaxMindDB(glTarPath, dir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mmdb, err := maxminddb.Open(mmdbFilename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// var record interface{}
	var record struct {
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

	networks := mmdb.Networks()
	for networks.Next() {
		subnet, err := networks.Network(&record)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		log.Printf("%s: %s/%s, %t %t", subnet.String(), record.Country.IsoCode, record.RegisteredCountry.IsoCode, record.Traits.IsAnonymousProxy, record.Traits.IsSatelliteProvider)
	}
	if networks.Err() != nil {
		log.Fatal(networks.Err())
	}
}