package i2c

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	tgz, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer tgz.Close()
	gz, err := gzip.NewReader(tgz)
	if err != nil {
		return
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, e := tr.Next()
		if err == io.EOF {
			break
		}
		if e != nil {
			err = e
			return
		}
		f := path.Base(header.Name)
		fInfo := header.FileInfo()
		if !fInfo.Mode().IsRegular() || filepath.Ext(f) != ".mmdb" {
			continue
		}
		mmdbFilename = path.Join(baseDir, f)
		log.Printf("Found %s", f)
		out, e := os.Create(mmdbFilename)
		if e != nil {
			err = e
			return
		}
		defer out.Close()
		n, e := io.Copy(out, tr)
		if e != nil {
			err = e
			return
		}
		if n != fInfo.Size() {
			err = fmt.Errorf("File size is %d but wrote %d", fInfo.Size(), n)
			return
		}
		log.Printf("Extracted %s", mmdbFilename)
		filename = mmdbFilename
		return
	}
	if mmdbFilename == "" {
		err = errors.New("Could not find .mmdb file in archive")
	}
	return
}

// GetMMDBFile extracts the mmdb file. The provided directory serves as a working directory and is intended to be used with a directory created using ioutil.TempDir
func GetMMDBFile(dir string) (filename string, err error) {
	glTarPath, err := downloadDB(geoliteURL, dir)
	if err != nil {
		log.Fatal(err)
		return
	}
	filename, err = extractMaxMindDB(glTarPath, dir)
	return
}

// GetRIRFiles downloads all RIR files concurrently and returns the list of downloaded files. The provided directory serves as a working directory and is intended to be used with a directory created using ioutil.TempDir
func GetRIRFiles(dir string) (filenames []string, err error) {
	filename := make(chan string, len(rirURLs))
	errs := make(chan error, len(rirURLs))
	for _, rirURL := range rirURLs {
		go func(url string) {
			f, err := downloadDB(url, dir)
			if err != nil {
				errs <- err
				filename <- ""
				return
			}
			filename <- f
			errs <- nil
		}(rirURL)
	}
	var errMsg string
	for i := 0; i < len(rirURLs); i++ {
		filenames = append(filenames, <-filename)
		if err := <-errs; err != nil {
			errMsg = fmt.Sprintf("%s\n%s", errMsg, err.Error())
		}
	}
	if errMsg != "" {
		err = errors.New(errMsg)
	}
	return
}

func MoveFile(srcPath, destPath string) (err error) {
	inFile, err := os.Open(srcPath)
	if err != nil {
		return
	}
	defer inFile.Close()
	outFile, err := os.Create(destPath)
	if err != nil {
		return
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return
	}
	err = os.Remove(srcPath)
	return
}

// WriteI2C writes the IP-to-coutnry mappings to file
func WriteI2C(entries map[string]string, outfile, tmpDir string) (err error) {
	tmpFilePath := path.Join(tmpDir, "i2c.conf")

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
	err = MoveFile(tmpFilePath, outfile)
	return
}
