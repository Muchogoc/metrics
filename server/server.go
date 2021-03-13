package main

import (
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"

	"contrib.go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func main() {
	// Firstly, we'll register ochttp Server views.
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		log.Fatalf("Failed to register server views for HTTP metrics: %v", err)
	}

	// Enable observability to extract and examine stats.
	enableObservabilityAndExporters()

	// The handler containing your business logic to process requests.
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Consume the request's body entirely.
		io.Copy(ioutil.Discard, r.Body)

		// Generate some payload of random length.
		res := strings.Repeat("a", rand.Intn(99971)+1)

		// Sleep for a random time to simulate a real server's operation.
		time.Sleep(time.Duration(rand.Intn(977)+1) * time.Millisecond)

		// Finally write the body to the response.
		w.Write([]byte("Hello, World! " + res))
	})

	och := &ochttp.Handler{
		Handler: originalHandler, // The handler you'd have used originally
	}
	if err := http.ListenAndServe(":8000", och); err != nil {
		panic(err)
	}
}

func enableObservabilityAndExporters() {
	// // Stats exporter: Prometheus
	// pe, err := prometheus.NewExporter(prometheus.Options{
	// 	Namespace: "ochttp_tutorial_server",
	// })
	// if err != nil {
	// 	log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	// }

	// view.RegisterExporter(pe)
	// go func() {
	// 	mux := http.NewServeMux()
	// 	mux.Handle("/metrics", pe)
	// 	log.Fatal(http.ListenAndServe(":8000", mux))
	// }()

	// Trace exporter: Zipkin
	localEndpoint, err := openzipkin.NewEndpoint("ochttp_tutorial_server", "localhost:5454")
	if err != nil {
		log.Fatalf("Failed to create the local zipkinEndpoint: %v", err)
	}
	reporter := zipkinHTTP.NewReporter("http://localhost:9411/api/v2/spans")
	ze := zipkin.NewExporter(reporter, localEndpoint)
	trace.RegisterExporter(ze)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}
