[![CircleCI](https://circleci.com/gh/Hsn723/nginx-i2c.svg?style=svg)](https://circleci.com/gh/Hsn723/nginx-i2c)
[![GoDoc](https://godoc.org/github.com/Hsn723/nginx-i2c?status.svg)](https://godoc.org/github.com/Hsn723/nginx-i2c)
[![Go Report Card](https://goreportcard.com/badge/github.com/Hsn723/nginx-i2c)](https://goreportcard.com/report/github.com/Hsn723/nginx-i2c)

# nginx-i2c
nginx-i2c generates IP to country mappings for use in [ngx_http_geo_module](https://nginx.org/en/docs/http/ngx_http_geo_module.html) using the CIDR format. It supports IPv4 and IPv6 subnets, with the option to only use IPv4 subnets for IPv4-only servers. The [MaxMind GeoIP2](https://dev.maxmind.com/geoip/) database is used and complemented with data from [AFRINIC](https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest), [APNIC](https://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest), [ARIN](https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest). [LACNIC](https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest) and [RIPE](https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest).

## Usage
Compile with `go build .`

```
Usage:
  nginx-i2c [flags]

Flags:
  -h, --help             help for nginx-i2c
  -4, --ipv4-only        only include IPv4 ranges
  -o, --outfile string   specify output file path (default "./ip2country.conf")
```

Use with ngx_http_geo_module. No `default` country is specified. Choose your own default country by adding the relevant setting before including `ip2country.conf`.
```
geo $country {
    default        JP;
    include        conf/ip2country.conf;
    # extra rules here
}
```

## Under consideration
- Include/exclude countries (mutually exclusive)
- IP range output?