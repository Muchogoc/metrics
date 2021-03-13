package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"
	"moul.io/http2curl"

	"contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func main() {
	// Firstly, we'll register ochttp Client views
	if err := view.Register(ochttp.DefaultClientViews...); err != nil {
		log.Fatalf("Failed to register client views for HTTP metrics: %v", err)
	}

	// For tracing, let's always sample for the purposes of this demo
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// Enable observability to extract and examine traces and metrics.
	enableObservabilityAndExporters()

	client := &http.Client{Transport: &ochttp.Transport{}}
	i := uint64(0)

	// Then finally do the work every 5 seconds.
	for {
		i += 1
		log.Printf("Performing fetch #%d", i)
		ctx, span := trace.StartSpan(context.Background(), fmt.Sprintf("Fetch-%d", i))
		doWork(ctx, client)
		span.End()

		<-time.After(5 * time.Second)
	}

}

func doWork(ctx context.Context, client *http.Client) {
	body := strings.NewReader(strings.Repeat("a", rand.Intn(777)+1))

	// req, _ := http.NewRequest("GET", "https://opencensus.io/", nil)
	req, _ := http.NewRequest("POST", "http://127.0.0.1:8000", body)

	// It is imperative that req.WithContext is used to
	// propagate context and use it in the request.
	req = req.WithContext(ctx)

	command, _ := http2curl.GetCurlCommand(req)
	fmt.Println(command)

	// Now make the request to the remote end.
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to make the request: %v", err)
		return
	}

	// Consume the body and close it.
	io.Copy(ioutil.Discard, res.Body)
	_ = res.Body.Close()

}

func enableObservabilityAndExporters() {
	// Stats exporter: Prometheus
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "ochttp_tutorial_client",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		log.Fatal(http.ListenAndServe(":8888", mux))
	}()

	// Trace exporter: Zipkin
	localEndpoint, _ := openzipkin.NewEndpoint("ochttp_tutorial_client", "localhost:0")
	reporter := zipkinHTTP.NewReporter("http://localhost:9411/api/v2/spans")
	ze := zipkin.NewExporter(reporter, localEndpoint)
	trace.RegisterExporter(ze)
}
