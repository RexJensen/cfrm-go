package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

//go:embed index.html
var indexHTML []byte

func main() {
	outputPath := flag.String("output", "output/output.json", "path to solver output JSON")
	port := flag.Int("port", 8080, "port to serve on")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(*outputPath)
		if err != nil {
			http.Error(w, "could not read output: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("CFRM Solver UI\n")
	fmt.Printf("Reading output from: %s\n\n", *outputPath)
	fmt.Printf("  Local:   http://localhost%s\n", addr)
	for _, ip := range localIPs() {
		fmt.Printf("  Network: http://%s%s  ← open this on your phone\n", ip, addr)
	}
	fmt.Println()
	log.Fatal(http.ListenAndServe(addr, nil))
}

// localIPs returns all non-loopback IPv4 addresses on this machine.
func localIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}
