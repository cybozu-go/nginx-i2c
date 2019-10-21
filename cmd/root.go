package cmd

import (
	"github.com/Hsn723/nginx-i2c/i2c"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

var (
	ipv4Only bool
	outfile  string
	rootCmd  = &cobra.Command{
		Use:   "nginx-i2c",
		Short: "nginx-i2c generates an IP-to-country mapping file for ngx_http_geo_module",
		Run:   rootMain,
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&ipv4Only, "ipv4-only", "4", false, "only include IPv4 ranges")
	rootCmd.PersistentFlags().StringVarP(&outfile, "outfile", "o", "./ip2country.conf", "specify output file path")
	// TODO: exclude countries
}

func rootMain(cmd *cobra.Command, args []string) {
	workDir, err := ioutil.TempDir("", "i2c")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(workDir)
	log.Printf("Created temporary directory %s", workDir)
	mmdbFilename, err := i2c.GetMMDBFile(workDir)
	if err != nil {
		log.Fatal(err)
	}
	mmdb, err := i2c.GetDBReader(mmdbFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer mmdb.Close()

	entries := map[string]string{}

	if err := i2c.GetMMDBSubnets(mmdb, entries, ipv4Only); err != nil {
		log.Fatal(err)
	}

	rirFiles, err := i2c.GetRIRFiles(workDir)
	if err != nil {
		log.Fatal(err)
	}
	if err := i2c.AppendAllRIRSubnets(mmdb, entries, rirFiles, ipv4Only); err != nil {
		log.Fatal(err)
	}

	if err := i2c.WriteI2C(entries, outfile, workDir); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote .conf file to %s", outfile)
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
