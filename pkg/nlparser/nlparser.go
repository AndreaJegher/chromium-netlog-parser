package nlparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type (
	// EventPart network log event fragment
	EventPart struct {
		// Possible params
		//scts []interface {}
		//proxy string
		//ndots float64
		//upload_id string
		//public_key_hashes []interface {}
		//description string
		//cached bool
		//timeout float64
		//unhandled_options bool
		//attempts float64
		//privacy_mode float64
		//is_chunked bool
		//rotate bool
		//group_id string
		//entry_hash string
		//expiration string
		//id string
		//version float64
		//index float64
		//pac_string string
		//using_quic bool
		//priority string
		//ipv6_available bool
		//bytes string
		//has_md5 bool
		//parent_stream_id float64
		//host string
		//is_ack bool
		//stream_id float64
		//proto string
		//address_family float64
		//address string
		//byte_count float64
		//cipher_suite float64
		//load_flags float64
		//current_position float64
		//truncate bool
		//has_md2 bool
		//new_config map[string]interface {}
		//url string
		//is_speculative bool
		//addresses []interface {}
		//ct_compliance_status string
		//size float64
		//search []interface {}
		//attempt_number float64
		//backup_job bool
		//transport_rtt_ms float64
		//is_issued_by_known_root bool
		//embedded_scts string
		//location string
		//doh_servers []interface {}
		//method string
		//source_address string
		//scts_from_ocsp_response string
		//build_timely bool
		//fin bool
		//use_local_ipv6 bool
		//is_preconnect bool
		//has_md4 bool
		//total_size float64
		//offset float64
		//expect_spdy bool
		//protocol string
		//error_code string
		//unique_id float64
		//os_error_string string
		//http_rtt_ms float64
		//allow_cached_response bool
		//certificates []interface {}
		//verified_cert map[string]interface {}
		//next_proto string
		//downstream_throughput_kbps float64
		//window_size float64
		//headers []interface {}
		//canonical_name string
		//domain string
		//type string
		//reason string
		//os_error float64
		//effective_connection_type string
		//address_list []interface {}
		//is_resumed bool
		//has_priority bool
		//persistent_store bool
		//cert_status float64
		//delta float64
		//delta_window_size float64
		//should_wait bool
		//settings []interface {}
		//certificate map[string]interface {}
		//filters string
		//persistence bool
		//scts_from_tls_extension string
		//key string
		//proxy_server string
		//original_url string
		//servers []interface {}
		//buf_len float64
		//source_dependency map[string]interface {}
		//net_error float64
		//is_issued_by_additional_trust_anchor bool
		//weight float64
		//exclusive bool
		//bytes_copied float64
		//num_hosts float64
		//append_to_multi_label_name bool
		//nameservers []interface {}
		//value float64
		Params map[string]interface{} `json:"params,omitempty"`
		Phase  int                    `json:"phase"`
		Source source                 `json:"source"`
		Time   string                 `json:"time"`
		Type   int                    `json:"type"`
	}

	source struct {
		ID   int `json:"id"`
		Type int `json:"type"`
	}

	// Event network log event
	Event struct {
		ID    int
		Type  string
		Parts []EventPart
	}

	// NetLog struct rapresenting a net log generated by chrome browser
	NetLog struct {
		Events        map[int]Event
		EventTypesMap map[int]string
	}

	// Redirection http redirection
	Redirection struct {
		From   string
		To     string
		Status int
		Time   int64
	}

	// URLRequest find requested urls
	URLRequest struct {
		URL  string
		Time int64
	}

	// DNSQuery queries to dns servers
	DNSQuery struct {
		Host        string
		AddressList []net.IP
		Time        int64
	}

	// Connection socket connection
	Connection struct {
		Type        string
		Source      string
		Destination string
	}

	// Source file used to render the web page (html, js, css, etc...)
	Source struct {
		ResourceName       string
		Base64EncodedBytes []string
	}
)

const (
	// URLRequestType type of an event whene a url request was made
	URLRequestType = "URL_REQUEST"
)

func stringTimeStampToUINT64(s string) int64 {
	t, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return t
}

// FindDNSQueries find asked dns queries
func (n NetLog) FindDNSQueries() []DNSQuery {
	r := []DNSQuery{}
	for _, e := range n.Events {
		if e.Type == "HOST_RESOLVER_IMPL_JOB" {
			req := DNSQuery{}
			for _, p := range e.Parts {
				h := p.Params["host"]
				if h != nil {
					req.Host = h.(string)
					req.Time = stringTimeStampToUINT64(p.Time)
				}
				al := p.Params["address_list"]
				if al != nil {
					for _, addr := range al.([]interface{}) {
						saddr := strings.Split(addr.(string), ":")[0]
						req.AddressList = append(req.AddressList, net.ParseIP(saddr))
					}
				}
			}
			r = append(r, req)
		}
	}
	return r
}

// FindURLRequests find url requsted
func (n NetLog) FindURLRequests() []URLRequest {
	r := []URLRequest{}
	for _, e := range n.Events {
		if e.Type == URLRequestType {
			for _, p := range e.Parts {
				req := URLRequest{}
				u := p.Params["url"]
				if u == nil {
					continue
				}
				req.URL = u.(string)
				req.Time = stringTimeStampToUINT64(p.Time)
				r = append(r, req)
				break
			}
		}
	}
	return r
}

// FindRedirections find http rederections
func (n NetLog) FindRedirections() []Redirection {
	r := []Redirection{}
	for _, e := range n.Events {
		if e.Type == URLRequestType {
			red := Redirection{}
			for _, p := range e.Parts {
				u := p.Params["url"]
				if u != nil && red.From == "" {
					red.From = u.(string)
				}

				h := p.Params["headers"]
				if h != nil && !strings.Contains(fmt.Sprint(h.([]interface{})[0]), "method") {
					red.Time = stringTimeStampToUINT64(p.Time)
					for _, s := range h.([]interface{}) {
						switch s.(type) {
						case string:
							if strings.Contains(s.(string), "302") {
								red.Status = 302
							}
							if strings.Contains(s.(string), "301") {
								red.Status = 301
							}
							if strings.Contains(s.(string), "location") {
								red.To = strings.Split(s.(string), " ")[1]
							}
						}
					}
				}
			}
			if red.From != "" && red.To != "" && red.Status != 0 {
				r = append(r, red)
			}
		}
	}
	return r
}

// FindSources search for sources
func (n NetLog) FindSources() []Source {
	res := []Source{}
	for _, e := range n.Events {
		s := Source{}
		if e.Type != URLRequestType {
			continue
		}
		for _, p := range e.Parts {
			u, ok := p.Params["url"]
			if ok {
				// in the case there are multiple urls it means
				// there were some redirection
				// so we take the last url in the chain of redirections
				s.ResourceName = fmt.Sprintf("%s", u)
			}
			// unencrypted data
			if p.Type == 111 {
				b, ok := p.Params["bytes"]
				if ok {
					s.Base64EncodedBytes = append(s.Base64EncodedBytes, fmt.Sprintf("%s", b))
				}
			}
		}
		if s.ResourceName != "" && len(s.Base64EncodedBytes) > 0 {
			res = append(res, s)
		}
	}
	return res
}

// FindOpenedSocket find opened socket events
func (n NetLog) FindOpenedSocket() []Connection {
	res := []Connection{}
	for _, e := range n.Events {
		if e.Type == "SOCKET" || e.Type == "UDP_SOCKET" {
			c := Connection{
				Source:      "UNKNOWN",
				Destination: "UNKNOWN",
				Type:        "TCP",
			}
			if e.Type == "UDP_SOCKET" {
				c.Type = "UDP"
			}
			for _, p := range e.Parts {
				u := p.Params["address"]
				if u != nil {
					c.Destination = fmt.Sprintf("%s", u)
				}
				s := p.Params["source_address"]
				if s != nil {
					c.Source = fmt.Sprintf("%s", s)
				}
			}
			res = append(res, c)
		}
	}
	return res
}

// FindDependenciesIDs find dependent event ids
func (e Event) FindDependenciesIDs() []int {
	out := []int{}
	for _, p := range e.Parts {
		a, ok := p.Params["source_dependency"]
		if ok {
			out = append(out, int(a.(map[string]interface{})["id"].(float64)))
		}
	}
	return out
}

// PrintEvent prints an event
func (e Event) String() string {
	s := fmt.Sprintf("ID: %d\nType: %s", e.ID, e.Type)
	for i, p := range e.Parts {
		s = fmt.Sprintf("%s\n    %d: %v", s, i, p)
	}
	return s
}

// ParseNetLog parse a net log
func ParseNetLog(file string) (NetLog, error) {
	var netlog NetLog
	netlog.EventTypesMap = make(map[int]string)
	netlog.Events = make(map[int]Event)

	f, err := os.Open(file)
	if err != nil {
		return netlog, err
	}

	headers := true
	s := bufio.NewScanner(f)
	buf := []byte{}
	// TODO: find buffer size dynamically
	s.Buffer(buf, 64*1024*3)
	for s.Scan() {
		line := s.Text()
		if headers {
			if strings.Contains(line, "\"events\"") {
				// headers are over
				// the events start on the next line
				headers = false
				continue
			}
			line = strings.TrimSuffix(line, ",") + "}"
			f := make(map[string]interface{})
			err := json.Unmarshal([]byte(line), &f)
			if err != nil {
				return netlog, err
			}
			for k, v := range f["constants"].(map[string]interface{})["logEventTypes"].(map[string]interface{}) {
				netlog.EventTypesMap[int(v.(float64))] = k
			}
			for k, v := range f["constants"].(map[string]interface{})["logSourceType"].(map[string]interface{}) {
				netlog.EventTypesMap[int(v.(float64))] = k
			}
		} else {
			p := EventPart{}
			line = strings.TrimSuffix(line, ",")
			err := json.Unmarshal([]byte(line), &p)
			if err != nil {
				// Last line have a trailing ]} instead of a ,
				line = strings.TrimSuffix(line, "]}")
				err = json.Unmarshal([]byte(line), &p)
				if err != nil {
					return netlog, err
				}
			}
			e, ok := netlog.Events[p.Source.ID]
			if ok {
				e.Parts = append(e.Parts, p)
				netlog.Events[p.Source.ID] = e
			} else {
				t, ok := netlog.EventTypesMap[p.Source.Type]
				if !ok {
					t = "UNKNOWN_TYPE"
				}
				e = Event{
					ID:    p.Source.ID,
					Type:  t,
					Parts: []EventPart{p},
				}
				netlog.Events[p.Source.ID] = e
			}
		}
	}
	return netlog, s.Err()
}
