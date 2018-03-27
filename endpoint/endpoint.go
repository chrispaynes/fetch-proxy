package endpoint

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Endpoint struct containing everything needed for a new endpoint
type Endpoint struct {
	Active     bool
	Address    *url.URL
	Proxy      *httputil.ReverseProxy
	Registered string
	Available  time.Time
}

// HealthCheck performs a basic http check based on a positive(<500) status code
func (e *Endpoint) HealthCheck(healthCheckURL string) {
	// previousStatus := e.Active
	// statusCode := 500
	// if resp, err := http.Get(e.Address.String() + "/" + healthCheckURL); err != nil {
	// 	// Something is up ... disable this endpoint
	// 	// e.Active = false
	// } else {
	// 	// Woot! Good to go ...
	// 	statusCode = resp.StatusCode
	// 	if resp.StatusCode >= 500 {
	// 		// e.Active = false
	// 	} else {
	// 		e.Active = true
	// 	}
	// }
	// log.WithFields(log.Fields{
	// 	"previous": previousStatus,
	// 	"current":  e.Active,
	// }).Debug(e.Registered)
	// if e.Active != previousStatus {
	// 	if e.Active {
	// 		// Whew, we came back online
	// 		log.WithFields(
	// 			log.Fields{
	// 				"URL": e.Address.String(),
	// 			}).Info("Up")
	// 		e.Available = time.Now()
	// 	} else {
	// 		// BOO HISS!
	// 		log.WithFields(
	// 			log.Fields{
	// 				"URL":    e.Address.String(),
	// 				"Status": statusCode,
	// 			}).Error("Down")
	// 	}
	// }
}

// NewEndpoint creates new endpoints to forward traffic to
func NewEndpoint(base, address string, checkHealth bool, healthURL string) (*Endpoint, error) {
	fmt.Printf("ADDRESS: %s\n", address)
	parsedAddress, err := url.Parse(address)
	if err != nil {
		log.WithFields(log.Fields{"url": parsedAddress}).Error("Problem parsing URL")
		return nil, errors.New("Problem parsing URL " + address)
	}
	e := &Endpoint{
		Address:    parsedAddress,
		Proxy:      httputil.NewSingleHostReverseProxy(parsedAddress),
		Active:     true,
		Available:  time.Now(),
		Registered: base,
	}
	// if checkHealth {
	// 	e.HealthCheck(healthURL)
	// } else {
	// Make it active regardless. Godspeed developer :D
	// e.Active = true
	// }
	return e, nil
}

//////////////////////////////////////

////////////////////////////////////////////////
// Store all of our endpoints
var endpoints map[string]*Endpoint
var endpointkeys sort.StringSlice

// passThrough takes in traffic on specific port and passes it through to the appropriate endpoint
func passThrough(w http.ResponseWriter, r *http.Request, defaultEndpoint string) {
	// remove www.
	if strings.HasPrefix(r.Host, "www.") {
		r.Host = strings.Replace(r.Host, "www.", "", 1)
	}

	fmt.Println("r.host:", r.Host)
	endpoint := siteKey(r.Host, defaultEndpoint)

	log.WithFields(
		log.Fields{
			"Request":   r.Host,
			"IP":        r.RemoteAddr,
			"Forwarded": endpoint,
		}).Info("New Request")

	// One quick sanity check before sending it on it's way
	if _, exists := endpoints[endpoint]; exists {
		fmt.Printf("ENDPOINTS %v", endpoints)
		endpoints[endpoint].Proxy.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Error 502 - Bad Gateway"))
	}
}

// FetchProxyStart creates and starts the proxy
func FetchProxyStart(httpPort int, secured, healthChecks bool, healthCheckURL, defaultEndpoint string) {
	log.WithFields(
		log.Fields{
			"port": httpPort,
		}).Info("Starting fetch proxy")

	// Start our healthchecks
	if healthChecks {
		// go HealthChecks(healthCheckURL)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		passThrough(w, r, defaultEndpoint)
	})

	http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
	fmt.Println("LISTENING AND SERVING", httpPort)

	// if !secured {
	// 	// Not secured, so lets just start a simple webserver
	// 	if err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil); err != nil {
	// 		log.Fatal(err.Error())
	// 		// shutdown.Now()
	// 	}
	// } else {
	// 	// start our letsencrypt SSL goodies
	// 	var m letsencrypt.Manager
	// 	if err := m.CacheFile("letsencrypt.cache"); err != nil {
	// 		log.Fatal(err)
	// 		// shutdown.Now()
	// 	}
	// 	log.Fatal(m.Serve())
	// }

}

// AddSite adds a new website to the proxy to be forwarded
func AddSite(base, address string, healthChecks bool, healthCheckURL string) error {
	// Check if endpoint already exists
	for _, item := range endpoints {
		if item.Registered == base && item.Address.String() == address {
			return nil
		}
	}

	// Construct the key so that you can sort by url base and time added
	urlbase := base

	// Remove any thing after the _ from the url
	if strings.Contains(urlbase, "_") {
		urlbase = urlbase[0:strings.Index(urlbase, "_")]
	}

	// // Remove any thing after the _ from the url
	// if strings.Contains(urlbase, "-") {
	// 	urlbase = urlbase[0:strings.Index(urlbase, "-")]
	// }

	key := urlbase + "-" + time.Now().Format("2006-01-02T15:04:05.000")

	// Add new endpoint
	ep, err := NewEndpoint(base, address, healthChecks, healthCheckURL)
	// need to verify there aren't name collisions
	if err == nil {
		// If it doesn't exist ...
		log.WithFields(log.Fields{
			"url":        address,
			"registered": base,
			"urlbase":    urlbase,
		}).Info("Registered endpoint")
		endpoints[key] = ep
		endpointkeys = append(endpointkeys, key)

		sort.Sort(sort.Reverse(endpointkeys))

		return nil
	}
	return err
}

// HealthChecks starts the background process for __all__ site health checks
func HealthChecks(healthCheckURL string) {
	// for {
	// 	<-time.After(10 * time.Second)
	// 	for key := range endpoints {
	// 		go endpoints[key].HealthCheck(healthCheckURL)
	// 	}
	// }
}

// Site key determines the endpoint to use based on the host
func siteKey(host, defaultEndpoint string) string {
	registered := ""
	// Grab the first key in the list that matches
	for _, key := range endpointkeys {
		b := endpoints[key].Registered

		// Allow for multiple containers with the same url
		if strings.Contains(b, "_") {
			b = b[0:strings.Index(b, "_")]
		}

		if strings.HasPrefix(defaultEndpoint, b) && endpoints[key].Active {
			defaultEndpoint = key
		}

		if strings.HasPrefix(host, b) && endpoints[key].Active {
			registered = key
			break
		}
	}

	if registered == "" {
		return defaultEndpoint
	}

	return registered
}

// init our maps
func init() {
	endpoints = make(map[string]*Endpoint)
}
