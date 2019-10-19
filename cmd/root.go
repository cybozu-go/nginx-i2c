package cmd

import (
	"fmt"
	"github.com/Hsn723/nginx-i2c/i2c"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

var (
	IPv4Only bool
	Outfile  string
	rootCmd  = &cobra.Command{
		Use:   "nginx-i2c",
		Short: "nginx-i2c generates an IP-to-country mapping file for ngx_http_geo_module",
		Run:   rootMain,
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&IPv4Only, "ipv4-only", "4", false, "only include IPv4 ranges (experimental)")
	rootCmd.PersistentFlags().StringVarP(&Outfile, "outfile", "o", "./ip2country.conf", "specify output file path")
	// TODO: exclude countries
}

func rootMain(cmd *cobra.Command, args []string) {
	workDir, err := ioutil.TempDir("", "i2c")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)
	log.Printf("Created temporary directory %s", workDir)
	mmdbFilename, err := i2c.GetMMDBFile(workDir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mmdb, err := i2c.GetDBReader(mmdbFilename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	entries := map[string]string{}

	if err := i2c.GetMMDBSubnets(mmdb, entries, IPv4Only); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	rirFiles, err := i2c.GetRIRFiles(workDir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	if err := i2c.AppendAllRIRSubnets(mmdb, entries, rirFiles, IPv4Only); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if err := i2c.WriteI2C(entries, Outfile, workDir); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Printf("Wrote .conf file to %s", Outfile)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
