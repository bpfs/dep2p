package main

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
)

const (
	pkgName        = "asnutil"
	ipv6OutputFile = "ipv6_asn_map.gen.go"
	ipv6listName   = "ipv6CidrToAsnPairList"
)

const defaultFile = "https://iptoasn.com/data/ip2asn-v6.tsv.gz"

func main() {
	// file with the ASN mappings for IPv6 CIDRs.
	// See ipv6_asn.tsv
	ipv6File := os.Getenv("ASN_IPV6_FILE")

	if len(ipv6File) == 0 {
		ipv6File = defaultFile
	}
	if strings.Contains(ipv6File, "://") {
		local, err := getMappingFile(ipv6File)
		if err != nil {
			panic(err)
		}
		ipv6File = local
	}

	ipv6CidrToAsnMap := readMappingFile(ipv6File)
	f, err := os.Create(ipv6OutputFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	writeMappingToFile(f, ipv6CidrToAsnMap, ipv6listName)
}

type listEntry struct {
	cidr, asn string
}

func writeMappingToFile(f *os.File, m map[string]string, listName string) {
	printf := func(s string, args ...interface{}) {
		_, err := fmt.Fprintf(f, s, args...)
		if err != nil {
			panic(err)
		}
	}
	printf("package %s\n\n", pkgName)
	printf("// Code generated by generate/main.go DO NOT EDIT\n")
	printf("var %s = [...]struct{ cidr, asn string }{", listName)
	l := make([]listEntry, 0, len(m))
	for k, v := range m {
		l = append(l, listEntry{cidr: k, asn: v})
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].cidr < l[j].cidr
	})
	for _, e := range l {
		printf("\n\t{\"%s\", \"%s\"},", e.cidr, e.asn)
	}
	printf("\n}\n")
}

func readMappingFile(path string) map[string]string {
	m := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			return m
		}

		startIP := record[0]
		endIP := record[1]
		asn := record[2]
		if asn == "0" {
			continue
		}

		s := net.ParseIP(startIP)
		e := net.ParseIP(endIP)
		if s.To16() == nil || e.To16() == nil {
			panic(errors.New("IP should be v6"))
		}

		prefixLen := zeroPrefixLen(XOR(s.To16(), e.To16()))
		cn := fmt.Sprintf("%s/%d", startIP, prefixLen)
		m[cn] = asn
	}
}

// Get a url, return file it's downloaded to. optionally gzip decode.
func getMappingFile(url string) (path string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	baseFile, err := os.CreateTemp("", "ip-map-download-*")
	if err != nil {
		return
	}
	defer baseFile.Close()
	_, err = io.Copy(baseFile, resp.Body)
	if err != nil {
		return
	}
	initBuf := make([]byte, 512)
	_, err = baseFile.ReadAt(initBuf, 0)
	if err != nil {
		return
	}
	if strings.Contains(http.DetectContentType(initBuf), "application/x-gzip") {
		// gunzip it.
		_, err = baseFile.Seek(0, io.SeekStart)
		if err != nil {
			return
		}
		var gzr *gzip.Reader
		gzr, err = gzip.NewReader(baseFile)
		if err != nil {
			return
		}
		var rawFile *os.File
		rawFile, err = os.CreateTemp("", "ip-map-download-*")
		if err != nil {
			return
		}
		defer os.Remove(baseFile.Name())
		defer rawFile.Close()
		_, err = io.Copy(rawFile, gzr)
		if err != nil {
			return
		}
		path = rawFile.Name()
		return
	}
	path = baseFile.Name()
	return
}

func zeroPrefixLen(id []byte) int {
	for i, b := range id {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(id) * 8
}

// XOR takes two byte slices, XORs them together, returns the resulting slice.
func XOR(a, b []byte) []byte {
	c := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}
	return c
}
