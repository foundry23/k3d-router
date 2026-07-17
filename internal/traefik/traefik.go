package traefik

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

type Route struct {
	Cluster               string
	LoadBalancerContainer string
	Hostnames             []string
}

type Config struct {
	HTTP *httpSection `json:"http,omitempty"`
	TCP  *tcpSection  `json:"tcp,omitempty"`
}

type httpSection struct {
	Routers  map[string]httpRouter  `json:"routers"`
	Services map[string]httpService `json:"services"`
}

type tcpSection struct {
	Routers  map[string]tcpRouter  `json:"routers"`
	Services map[string]tcpService `json:"services"`
}

type httpRouter struct {
	EntryPoints []string `json:"entryPoints"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
}

type tcpRouter struct {
	EntryPoints []string `json:"entryPoints"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	TLS         tcpTLS   `json:"tls"`
}

type tcpTLS struct {
	Passthrough bool `json:"passthrough"`
}

type httpServer struct {
	URL string `json:"url"`
}

type tcpServer struct {
	Address string `json:"address"`
}

type httpService struct {
	LoadBalancer httpLoadBalancer `json:"loadBalancer"`
}

type httpLoadBalancer struct {
	Servers []httpServer `json:"servers"`
}

type tcpService struct {
	LoadBalancer tcpLoadBalancer `json:"loadBalancer"`
}

type tcpLoadBalancer struct {
	Servers []tcpServer `json:"servers"`
}

func BuildConfig(routes []Route) (Config, []string) {
	owner := map[string]string{}
	var warnings []string

	httpRouters := map[string]httpRouter{}
	httpServices := map[string]httpService{}
	tcpRouters := map[string]tcpRouter{}
	tcpServices := map[string]tcpService{}

	for _, route := range routes {
		var routable []string
		for _, hostname := range route.Hostnames {
			if hostname == "" {
				continue
			}
			if existing, taken := owner[hostname]; taken {
				warnings = append(warnings, fmt.Sprintf(
					"hostname %q already routed to %q — ignoring it in %q",
					hostname, existing, route.Cluster,
				))
				continue
			}
			owner[hostname] = route.Cluster
			routable = append(routable, hostname)
		}
		if len(routable) == 0 {
			continue
		}
		sort.Strings(routable)

		httpParts := make([]string, len(routable))
		tcpParts := make([]string, len(routable))
		for i, hostname := range routable {
			httpParts[i] = fmt.Sprintf("Host(`%s`)", hostname)
			tcpParts[i] = fmt.Sprintf("HostSNI(`%s`)", hostname)
		}

		httpRouters[route.Cluster] = httpRouter{
			EntryPoints: []string{"web"},
			Rule:        strings.Join(httpParts, " || "),
			Service:     route.Cluster,
		}
		httpServices[route.Cluster] = httpService{
			LoadBalancer: httpLoadBalancer{
				Servers: []httpServer{{URL: fmt.Sprintf("http://%s:80", route.LoadBalancerContainer)}},
			},
		}
		tcpRouters[route.Cluster] = tcpRouter{
			EntryPoints: []string{"websecure"},
			Rule:        strings.Join(tcpParts, " || "),
			Service:     route.Cluster,
			TLS:         tcpTLS{Passthrough: true},
		}
		tcpServices[route.Cluster] = tcpService{
			LoadBalancer: tcpLoadBalancer{
				Servers: []tcpServer{{Address: fmt.Sprintf("%s:443", route.LoadBalancerContainer)}},
			},
		}
	}

	cfg := Config{}
	if len(httpRouters) > 0 {
		cfg.HTTP = &httpSection{Routers: httpRouters, Services: httpServices}
		cfg.TCP = &tcpSection{Routers: tcpRouters, Services: tcpServices}
	}
	return cfg, warnings
}

func (c Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

func Parse(data []byte) (Config, error) {
	var c Config
	err := yaml.Unmarshal(data, &c)
	return c, err
}

var hostRuleRE = regexp.MustCompile("`([^`]+)`")

func (c Config) Hosts() map[string]string {
	routes := map[string]string{}
	if c.HTTP == nil {
		return routes
	}
	for cluster, router := range c.HTTP.Routers {
		for _, match := range hostRuleRE.FindAllStringSubmatch(router.Rule, -1) {
			routes[match[1]] = cluster
		}
	}
	return routes
}

type HostChange struct {
	Host    string
	Cluster string
}

func Diff(oldRoutes, newRoutes map[string]string) (added, removed []HostChange) {
	for host, cluster := range newRoutes {
		if oldRoutes[host] != cluster {
			added = append(added, HostChange{Host: host, Cluster: cluster})
		}
	}
	for host, cluster := range oldRoutes {
		if newRoutes[host] != cluster {
			removed = append(removed, HostChange{Host: host, Cluster: cluster})
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i].Host < added[j].Host })
	sort.Slice(removed, func(i, j int) bool { return removed[i].Host < removed[j].Host })
	return added, removed
}
