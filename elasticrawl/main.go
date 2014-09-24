package main

import (
	"log"
	"flag"
	"time"
	"net/http"

	"github.com/fritz0705/filia"
	"github.com/belogik/goes"
	"gopkg.in/fatih/set.v0"
)

func main() {
	var (
		flJobs = flag.Int("jobs", 1, "Number of crawler jobs")
		flHost = flag.String("es-host", "localhost", "Elasticsearch host")
		flPort = flag.String("es-port", "9200", "Elasticsearch port")
		flIndex = flag.String("es-index", "filia", "Elasticsearch index")
		flTimeout = flag.Duration("timeout", 8 * time.Second, "Crawler timeout")
		flBuffer = flag.Int("buffer", 1<<16, "Internal channel buffer size")
		err error
	)

	flag.Parse()

	httpProto := filia.HTTPProto{
		Client: http.Client{
			Timeout: *flTimeout,
		},
	}

	crawler := filia.Crawler{
		Settings: filia.Settings{
			Decoders: filia.DefaultSettings.Decoders,
			Protos: map[string]filia.Proto{
				"http": httpProto,
				"https": httpProto,
			},
		},
		Queue: make(filia.StdCrawlerQueue, *flBuffer),
		Set: *set.New(),
		Output: make(chan filia.Document, *flBuffer),
	}

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
