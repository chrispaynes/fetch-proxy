package main

import (
	"flag"
	"net/http"
	"syscall"
	"time"

	"fetch-proxy/config"
	"fetch-proxy/docker"
	"fetch-proxy/endpoint"

	shutdown "github.com/kcmerrill/shutdown.go"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Setup some command line arguments
	port := flag.Int("port", 80, "The port in which the proxy will listen on")
	containerized := flag.Bool("containerized", false, "Is fetch-proxy running in a container?")
	insecure := flag.Bool("insecure", false, "Should use HTTP or HTTPS? HTTP works great for dev envs")
	disableHealthChecks := flag.Bool("disable-healthchecks", true, "disable health checks for dev envs")
	healthCheckURL := flag.String("healthcheck", "?health", "The url to be used for healthchecks")
	dev := flag.Bool("dev", true, "Disable health checks and HTTPS for dev envs")
	// timeout := flag.Int("response-timeout", 10, "The response timeout for the proxy")
	// default to 0 timeout so requests never timeout
	// let's make it clear that this is in seconds
	timeout := flag.Int("response-timeout", 0, "The response timeout (in seconds) for the proxy")
	defaultEndpoint := flag.String("default", "__default", "The default endpoint fetch-proxy uses when requested endpoing isn't found")
	cfg := flag.String("config", "", "Location for the configuration file you want to use")

	flag.Parse()

	// Disable ssl/tls and health checks in dev mode
	if *dev {
		*disableHealthChecks = true
		*insecure = true
	}

	// Set a global timeout
	// ?Why would we want this to be global when containers can have a variety of latencies

	// Timeout specifies a time limit for requests made by this
	// Client. The timeout includes connection time, any
	// redirects, and reading the response body. The timer remains
	// running after Get, Head, Post, or Do return and will
	// interrupt reading of the Response.Body.
	//
	// A Timeout of zero means no timeout.
	http.DefaultClient.Timeout = time.Duration(*timeout) * time.Second

	// Start our proxy on the specified port
	go endpoint.FetchProxyStart(*port, !*insecure, !*disableHealthChecks, *healthCheckURL, *defaultEndpoint)

	go docker.ContainerWatch(*containerized, !*disableHealthChecks, *healthCheckURL, *port)

	if *cfg != "" {
		go config.ConfigWatch(*cfg, *containerized, !*disableHealthChecks, *healthCheckURL)
	}

	// No need to shutdown the application _UNLESS_ we catch it
	shutdown.WaitFor(syscall.SIGINT, syscall.SIGTERM)
	log.Info("Shutting down ... ")

}
