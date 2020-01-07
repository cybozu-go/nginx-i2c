package cmd

import (
	"errors"
	"github.com/cybozu-go/nginx-i2c/i2c"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

var (
	lowercase        bool
	ipv4Only         bool
	outfile          string
	maxMindLicense   string
	includeCountries []string
	excludeCountries []string
	rootCmd          = &cobra.Command{
		Use:   "nginx-i2c",
		Short: "nginx-i2c generates an IP-to-country mapping file for ngx_http_geo_module",
		Args:  validateArgs,
		Run:   rootMain,
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&lowercase, "lower", "l", false, "output country codes in lowercase")
	rootCmd.PersistentFlags().BoolVarP(&ipv4Only, "ipv4-only", "4", false, "only include IPv4 ranges")
	rootCmd.PersistentFlags().StringVarP(&outfile, "outfile", "o", "./ip2country.conf", "specify output file path")
	rootCmd.PersistentFlags().StringSliceVarP(&includeCountries, "include", "i", []string{}, "countries whose subnets to include, cannot be used with --exclude")
	rootCmd.PersistentFlags().StringSliceVarP(&excludeCountries, "exclude", "e", []string{}, "countries whose subnets to exclude, cannot be used with --include")
	rootCmd.PersistentFlags().StringVarP(&maxMindLicense, "maxmind-token", "t", "", "token for use with MaxMind")

	rootCmd.AddCommand(versionCmd)
}

func validateCountries() error {
	if len(includeCountries) > 0 && len(excludeCountries) > 0 {
		return errors.New("--include cannot be used alongside --exclude")
	}
	if len(includeCountries) > 0 {
		for _, c := range includeCountries {
			if len(c) != 2 {
				return errors.New("only two letter country codes are accepted")
			}
		}
	}
	if len(excludeCountries) > 0 {
		for _, c := range excludeCountries {
			if len(c) != 2 {
				return errors.New("only two letter country codes are accepted")
			}
		}
	}
	return nil
}

func validateArgs(cmd *cobra.Command, args []string) error {
	return validateCountries()
}

func rootMain(cmd *cobra.Command, args []string) {
	workDir, err := ioutil.TempDir("", "i2c")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(workDir)
	log.Printf("Created temporary directory %s", workDir)
	mmdbFilename, err := i2c.GetMMDBFile(workDir, maxMindLicense)
	if err != nil {
		log.Fatal(err)
	}
	mmdb, err := i2c.GetDBReader(mmdbFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer mmdb.Close()

	entries := map[string]string{}
	includes := i2c.CountrySliceToMap(includeCountries)
	excludes := i2c.CountrySliceToMap(excludeCountries)

	if err := i2c.GetMMDBSubnets(mmdb, entries, ipv4Only, lowercase, includes, excludes); err != nil {
		log.Fatal(err)
	}

	rirFiles, err := i2c.GetRIRFiles(workDir)
	if err != nil {
		log.Fatal(err)
	}
	if err := i2c.AppendAllRIRSubnets(mmdb, entries, rirFiles, ipv4Only, lowercase, includes, excludes); err != nil {
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
