package main

import (
	"log"
	"flag"

	"github.com/fritz0705/filia"
	"github.com/belogik/goes"
)

func main() {
	var (
		flJobs = flag.Int("jobs", 1, "Number of crawler jobs")
		flHost = flag.String("es-host", "localhost", "Elasticsearch host")
		flPort = flag.String("es-port", "9200", "Elasticsearch port")
		flIndex = flag.String("es-index", "filia", "Elasticsearch index")
		err error
		crawler = filia.DefaultCrawler
	)

	crawler.Queue = make(filia.StdCrawlerQueue, 1 << 16)

	flag.Parse()

	esConn := goes.NewConnection(*flHost, *flPort)
	esConn.CreateIndex(*flIndex, nil)

	go crawler.Emit(flag.Args()...)

	for n := 0; n < *flJobs; n++ {
		go crawler.Crawl()
	}

	for doc := range crawler.Output {
		esDoc := goes.Document{
			Index: *flIndex,
			Type: "document",
			Fields: doc,
		}

		_, err = esConn.Index(esDoc, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}
