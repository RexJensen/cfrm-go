package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
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
	fmt.Printf("Serving CFRM UI at http://localhost%s\n", addr)
	fmt.Printf("Reading output from: %s\n", *outputPath)
	log.Fatal(http.ListenAndServe(addr, nil))
}
