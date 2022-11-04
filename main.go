package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/pexni/xhttp"
)

const (
	ContainerEventRename  = "rename"
	ContainerEventStart   = "start"
	ContainerEventUpdate  = "update"
	ContainerEventPause   = "pause"
	ContainerEventUnpause = "unpause"
	ContainerEventRestart = "restart"
	ContainerEventKill    = "kill"
	ContainerEventStop    = "stop"
	ContainerEventDie     = "die"

	DefaultDockerNetworkHost = "172.17.0.1"
)

var (
	DockerClient    client.APIClient
	ContainerRoutes = make(map[string]uint16, 0)
	ContainerRWLock = &sync.RWMutex{}
	ctx             = context.Background()
)

func init() {
	var err error
	// init docker client
	DockerClient, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
}

func main() {
	// listen docker containers and update the ContainerRoutes Map
	go ListenDockerContainers()
	ServeHttpProxy()
}

func ServeHttpProxy() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hostArr := strings.Split(r.Host, ".")
		if len(hostArr) == 2 || len(hostArr) > 3 {
			xhttp.Ok(w, 200, "success", nil)
			return
		}
		// get the service name
		serviceName := hostArr[0]
		// get the service post by service name
		port := ContainerRoutes[serviceName]
		targetHost := fmt.Sprintf("http://%s:%d", DefaultDockerNetworkHost, port)
		// new proxy
		proxy, err := NewProxy(targetHost)
		if err != nil {
			xhttp.BadRequest(w, 400, err.Error(), nil)
			return
		}
		proxy.ServeHTTP(w, r)
	})
	fmt.Println("http proxy run")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func ListenDockerContainers() {
	updateContainerRoutes()
	ctx := context.Background()
	// listen docker client events
	msg, errs := DockerClient.Events(ctx, types.EventsOptions{})
	go func() {
		for e := range errs {
			log.Println(e)
		}
	}()
	// read messages
	go readMessages(msg)
	// keep running
	for {
		time.Sleep(1 * time.Minute)
	}
}

func readMessages(msg <-chan events.Message) {
	for m := range msg {
		if m.Type == "container" {
			switch m.Action {
			// those actions need to update container routes
			case
				ContainerEventRename,
				ContainerEventStart,
				ContainerEventUpdate,
				ContainerEventPause,
				ContainerEventUnpause,
				ContainerEventRestart,
				ContainerEventKill,
				ContainerEventStop,
				ContainerEventDie:
				updateContainerRoutes()
				log.Println(m.Action)
			}
		}
	}
}

func updateContainerRoutes() {
	// get all containers
	containers, err := DockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return
	}
	// RW lock
	ContainerRWLock.Lock()
	for _, container := range containers {
		for _, name := range container.Names {
			nameArr := strings.Split(name, "/")
			name = nameArr[len(nameArr)-1]
			for _, port := range container.Ports {
				// map the service name and local public port
				ContainerRoutes[name] = port.PublicPort
			}
		}
	}
	// RW unlock
	ContainerRWLock.Unlock()
}

func NewProxy(targetHost string) (*httputil.ReverseProxy, error) {
	targetUrl, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(targetUrl), nil
}
