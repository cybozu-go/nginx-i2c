package i2c

import (
	"errors"
	"github.com/mholt/archiver"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// Record describes a MaxMind DB record entry with only Country fields
type Record struct {
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	RegisteredCountry struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"registered_country"`
	IsAnonymousProxy    bool `maxminddb:"is_anonymous_proxy"`
	IsSatelliteProvider bool `maxminddb:"is_satellite_provider"`
}

func downloadDB(url string, baseDir string) (filename string, err error) {
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

// GetMMDBFile extracts the mmdb file from the archived located at the givel url. The provided directory serves as a working directory and is intended to be used with a directory created using ioutil.TempDir
func GetMMDBFile(url string, dir string) (filename string, err error) {
	glTarPath, err := downloadDB(url, dir)
	if err != nil {
		log.Fatal(err)
		return
	}
	filename, err = extractMaxMindDB(glTarPath, dir)
	return
}
