package meetup_client_go

import (
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"
	"net/url"
	"errors"
	"github.com/peterhellberg/link"
	"strconv"
	"time"
	_ "strings"
)

const API_BASE = "https://api.meetup.com"
const CATEGORIES = "/2/categories"
const VENUES = "/venues"
const V2_EVENTS = "/2/events"
const V3_EVENTS = "/events"
const MAX_VENUES_REQ = 50
const MAX_EVENTS_REQ = 50

// Error messages.
const HTTP_REQ_FAILED = "HTTP Request failed."
const JSON_UNMARSHAL_FAILED = "JSON Unmarshal failed."
const FAILED_TO_READ_RESP = "Failed to read response."


type JsonResponse map[string] interface{}
type JsonArray [] interface{}

type MeetupClient struct {
	api_key string
	rpm int
	sleep int
	rate_limit_set bool
	throttle bool
}

var logErr = func(v ...interface{}) {
	log.Println("Error: ", v[0], " Info: ", v[1])
}

var logReq = func(v ...string) {
	log.Println("Method: ", v[0], "Path: ", v[1], "Status: ", v[2])
}

func NewMeetupClient(apiKey string) (*MeetupClient) {
	if len(apiKey) == 0 {
		log.Println("API Key not specified.")
		return nil
	}
	c := new (MeetupClient)
	c.api_key = apiKey
	c.rpm = 0
	c.sleep = 0
	c.rate_limit_set = false
	c.throttle = false
	return c
}

func (c *MeetupClient) throttle_if_required(head http.Header) {

	var limit_rem, limit, window_rem int
	var err error

	if limit_rem, err = strconv.Atoi(head.Get("X-RateLimit-Remaining")); err != nil {
		log.Fatal("Error parsing int")
	}
	log.Println("API-Limit remaining", limit_rem)
	if window_rem, err = strconv.Atoi(head.Get("X-RateLimit-Remaining")); err != nil {
		log.Fatal("Error parsing int")
	}
	log.Println("API-Limit window rem", window_rem)
	if limit, err = strconv.Atoi(head.Get("X-RateLimit-Limit")); err != nil {
		log.Fatal("Error parsing int")
	}
	log.Println("API-Limit limit", limit)

	if float64(limit - limit_rem) > float64(limit/2) {
		c.throttle = true
		c.sleep = window_rem/2
		log.Println("Throttling...", c.sleep, " secs")
	}
}

func (c *MeetupClient) doV2Get(req_url *url.URL) (results JsonArray, err_out error) {

	// Fill credentials
	params := req_url.Query()
	if params["sign"] == nil {
		params.Set("sign", "true")
	}
	if params["key"] == nil {
		params.Set("key", c.api_key)
	}

	if res, err := http.Get(req_url.String()); err != nil {
		err_out = err
		logErr(err_out, HTTP_REQ_FAILED)
	} else {
		logReq(res.Request.Method, res.Request.URL.String(), res.Status)
		c.throttle_if_required(res.Header)
		if body, e := ioutil.ReadAll(res.Body); e != nil {
			err_out = e
			logErr(err_out, FAILED_TO_READ_RESP)
		} else {
			var resp JsonResponse
			err_out = json.Unmarshal(body, &resp)
			if err_out != nil {
				logErr(err_out, JSON_UNMARSHAL_FAILED)
			} else {
				results = JsonArray(resp["results"].([]interface{}))
			}
		}
	}
	return
}

func (c *MeetupClient) doV3GetArray(req_url *url.URL) (results JsonArray, err_out error, redirect_url string) {

	redirect_url = ""

	// Fill credentials
	params := req_url.Query()
	if params["sign"] == nil {
		params.Set("sign", "true")
	}
	if params["key"] == nil {
		params.Set("key", c.api_key)
	}

	if len(params) > 0 {
		req_url.RawQuery = params.Encode()
	}

	if res, err := http.Get(req_url.String()); err != nil {
		err_out = err
		logErr(err_out, HTTP_REQ_FAILED)
	} else {
		logReq(res.Request.Method, res.Request.URL.String(), res.Status)
		c.throttle_if_required(res.Header)
		if body, e := ioutil.ReadAll(res.Body); e != nil {
			err_out = e
			logErr(err_out, FAILED_TO_READ_RESP)
		} else {
			err_out = json.Unmarshal(body, &results)
			if err_out != nil {
				logErr(err_out, JSON_UNMARSHAL_FAILED)
			}

			g := link.ParseResponse(res)
			log.Println("Links size:", len(g))
			log.Println("Results size:", len(results))

			for i := range g {
				log.Println(g[i].Rel + g[i].URI)
				if g[i].Rel == "next" {
					redirect_url = g[i].URI
				}
			}
		}
	}
	return
}

func (c *MeetupClient) doV3GetObj(req_url *url.URL) (result JsonResponse, err_out error) {

	// Fill credentials
	params := req_url.Query()
	if params["sign"] == nil {
		params.Set("sign", "true")
	}
	if params["key"] == nil {
		params.Set("key", c.api_key)
	}

	if len(params) > 0 {
		req_url.RawQuery = params.Encode()
	}

	if res, err := http.Get(req_url.String()); err != nil {
		err_out = err
		logErr(err_out, HTTP_REQ_FAILED)
	} else {
		logReq(res.Request.Method, res.Request.URL.String(), res.Status)
		c.throttle_if_required(res.Header)
		if body, e := ioutil.ReadAll(res.Body); e != nil {
			err_out = e
			logErr(err_out, FAILED_TO_READ_RESP)
		} else {
			err_out = json.Unmarshal(body, &result)
			if err_out != nil {
				logErr(err_out, JSON_UNMARSHAL_FAILED)
			}
		}
	}
	return
}

func (c *MeetupClient) doRecursiveGet(req_url *url.URL) (results JsonArray, err_out error) {

	tmp_results, tmp_err, redirect := c.doV3GetArray(req_url)
	log.Println("Redirect:", redirect)
	for tmp_err == nil {
		results = append(results, tmp_results)
		if len(redirect) > 0 {
			redirect_url, _ := url.Parse(redirect)
			tmp_results, tmp_err, redirect = c.doV3GetArray(redirect_url)
		} else {
			return
		}
	}
	if tmp_err != nil {
		err_out = tmp_err
		logErr(err_out, "Error in recursive call.")
	}
	return
}

func (c *MeetupClient) SetRateLimit(set_rpm int) {
	c.rpm = set_rpm
	c.rate_limit_set = true
}

/*
 * V2 Categories.
 * Gets all meetup categories.
 *
 */
func (c *MeetupClient) GetCategories() (results JsonArray, err_out error) {

	url_str := API_BASE + CATEGORIES
	req_url, _ := url.Parse(url_str)
	results , err_out = c.doV2Get(req_url)
	return
}

/*
 * V3 Venues by group
 *
 * GET api.meetup.com/:urlname/venues
 * Input:
 *	urlname of group
 * Optional params:
 * 	page => no of results per page (seems like 50 is a limit here)
 *	offset => page offset. (1,2...)
 *	Other params are not yet supported.
 *
 * Output:
 *	JsonArray of venues.
 */
func (c *MeetupClient) GetVenuesByGroup(url_name string, params map[string] interface{}) (venues JsonArray, err_out error)  {

	if len(url_name) == 0 || (params["page"] != nil && params["page"].(int) > 50) {
		err_out = errors.New("Invalid request.")
		return
	}

	if req_url, e := url.Parse(API_BASE + "/" + url_name + VENUES); e != nil {
		log.Println("Error parsing group urlname provided.")
		err_out = e
	} else {
		u_params := req_url.Query()
		if params["page"] != nil {
			u_params.Set("page", strconv.Itoa(params["page"].(int)))
		} else {
			u_params.Set("page", strconv.Itoa(MAX_VENUES_REQ))
		}

		if params["offset"] != nil {
			u_params.Set("offset", strconv.Itoa(params["offset"].(int)))
		}
		req_url.RawQuery = u_params.Encode()
		venues, err_out, _ = c.doV3GetArray(req_url)
	}
	return
}

/*
 * V3 Venues by group (Multiple)
 * Will use rate limit to continuously make requests to get
 * all the venues in the group.
 *
 * GET api.meetup.com/:urlname/venues
 * Input:
 *	urlname of group
 * Optional params:
 * 	page => no of results per page (seems like 50 is a limit here)
 *	offset => page offset. (1,2...)
 *	Other params are not yet supported.
 *
 * Output:
 *	List of JsonArray of venues.
 */
func (c *MeetupClient) GetAllVenuesByGroup(url_name string) (venues_list JsonArray, err_out error)  {

	if len(url_name) == 0 {
		err_out = errors.New("Invalid request.")
		return
	}

	if req_url, e := url.Parse(API_BASE + "/" + url_name + VENUES); e != nil {
		log.Println("Error parsing group urlname provided.")
		err_out = e
	} else {
		done := false
		i := 0
		for done != true {
			u_params := req_url.Query()
			u_params.Set("page", strconv.Itoa(MAX_VENUES_REQ))
			u_params.Set("offset", strconv.Itoa(i))
			req_url.RawQuery = u_params.Encode()
			if venues, e, _ := c.doV3GetArray(req_url); e == nil {
				venues_list = append(venues_list, venues)
				if len(venues) < MAX_VENUES_REQ {
					done = true
				}
				if c.rate_limit_set {
					time.Sleep(time.Duration(60/c.rpm) * time.Second)
				} else if c.throttle {
					time.Sleep(time.Duration(c.sleep) * time.Second)
					c.throttle = false
				}
			} else {
				err_out = e
				logErr(err_out)
				done = true
			}
			i++
		}
	}
	return
}

/*
 * V3 Events by group
 *
 * GET api.meetup.com/:urlname/events
 * Input:
 *	urlname of group
 * Optional params:
 * 	page => no of results per page (seems like 50 is a limit here)
 *	offset => page offset. (1,2...)
 *	scroll => next_upcoming/recent_past
 *	status => Event status eg: upcoming
 *
 * If optional params are not supplied, this call will recursively call the url
 * provided in the HTTP Link header. (See API doc)
 * Output:
 *	JsonArray of events.
 */
func (c *MeetupClient) GetEventsByGroup(url_name string, params map[string] interface{}) (events JsonArray, err_out error)  {

	if len(url_name) == 0 || (params["page"] != nil && params["page"].(int) > 50) {
		err_out = errors.New("Invalid request.")
		return
	}

	if req_url, e := url.Parse(API_BASE + "/" + url_name + V3_EVENTS); e != nil {
		log.Println("Error parsing group urlname provided.")
		err_out = e
	} else {
		u_params := req_url.Query()
		if params["page"] != nil {
			u_params.Set("page", strconv.Itoa(params["page"].(int)))
		} else {
			u_params.Set("page", strconv.Itoa(MAX_EVENTS_REQ))
		}

		if params["offset"] != nil {
			u_params.Set("offset", strconv.Itoa(params["offset"].(int)))
		}

		if params["scroll"] != nil {
			u_params.Set("scroll", params["scroll"].(string))
		}

		if params["status"] != nil {
			u_params.Set("status", params["status"].(string))
		}
		req_url.RawQuery = u_params.Encode()
		events, err_out, _ = c.doV3GetArray(req_url)
	}
	return
}

/*
 * V2 Events
 *
 * GET api.meetup.com/2/events
 * Input:
 *	One of:
 *	event_id OR
 *	group_id OR
 *	group_urlname OR
 *	venue_id
 * Optional params:
 * 	page => no of results per page (seems like 50 is a limit here)
 *	offset => page offset. (1,2...)
 *	status => Event status eg: upcoming
 *
 * Output:
 *	JsonArray of events.
 */
func (c *MeetupClient) GetEvents(params map[string] interface{}) (events JsonArray, err_out error)  {

	if (params["event_id"] == nil && params["group_id"] == nil &&
		params["group_urlname"] == nil && params["venue_id"] == nil) ||
		(params["page"] != nil && params["page"].(int) > 50) {

		err_out = errors.New("Invalid request.")
		return
	}

	if req_url, e := url.Parse(API_BASE + V2_EVENTS); e != nil {
		log.Println("Error parsing group urlname provided.")
		err_out = e
	} else {
		u_params := req_url.Query()
		if params["page"] != nil {
			u_params.Set("page", strconv.Itoa(params["page"].(int)))
		} else {
			u_params.Set("page", strconv.Itoa(MAX_EVENTS_REQ))
		}

		if params["offset"] != nil {
			u_params.Set("offset", strconv.Itoa(params["offset"].(int)))
		}

		if params["status"] != nil {
			u_params.Set("status", params["status"].(string))
		}

		if params["event_id"] != nil {
			u_params.Set("event_id", params["event_id"].(string))
		}

		if params["group_id"] != nil {
			u_params.Set("group_id", params["group_id"].(string))
		}

		if params["group_urlname"] != nil {
			u_params.Set("group_urlname", params["group_urlname"].(string))
		}

		if params["venue_id"] != nil {
			u_params.Set("status", params["venue_id"].(string))
		}
		req_url.RawQuery = u_params.Encode()
		events, err_out = c.doV2Get(req_url)
	}
	return
}

/*
 * V2 All Events
 *
 * GET api.meetup.com/2/events
 * Input:
 *	One of:
 *	event_id OR
 *	group_id OR
 *	group_urlname OR
 *	venue_id
 * Optional params:
 * 	page => no of results per page (seems like 50 is a limit here)
 *	offset => page offset. (1,2...)
 *	status => Event status eg: upcoming
 *
 * Output:
 *	JsonArray of events.
 */
func (c *MeetupClient) GetAllEvents(params map[string] interface{}) (events_list JsonArray, err_out error)  {

	if (params["event_id"] == nil && params["group_id"] == nil &&
		params["group_urlname"] == nil && params["venue_id"] == nil) ||
		(params["page"] != nil && params["page"].(int) > 50) {

		err_out = errors.New("Invalid request.")
		return
	}

	if (params["event_id"] == nil && params["group_id"] == nil &&
		params["group_urlname"] == nil && params["venue_id"] == nil) ||
		(params["page"] != nil && params["page"].(int) > 50) {

		err_out = errors.New("Invalid request.")
		return
	}

	if req_url, e := url.Parse(API_BASE + V2_EVENTS); e != nil {
		log.Println("Error parsing group urlname provided.")
		err_out = e
	} else {
		done := false
		i := 0
		for done != true {
			u_params := req_url.Query()
			if params["status"] != nil {
				u_params.Set("status", params["status"].(string))
			}
			if params["event_id"] != nil {
				u_params.Set("event_id", params["event_id"].(string))
			}
			if params["group_id"] != nil {
				u_params.Set("group_id", params["group_id"].(string))
			}
			if params["group_urlname"] != nil {
				u_params.Set("group_urlname", params["group_urlname"].(string))
			}
			if params["venue_id"] != nil {
				u_params.Set("venue_id", params["venue_id"].(string))
			}
			u_params.Set("page", strconv.Itoa(MAX_EVENTS_REQ))
			u_params.Set("offset", strconv.Itoa(i))
			req_url.RawQuery = u_params.Encode()
			if events, e := c.doV2Get(req_url); e == nil {
				events_list = append(events_list, events)
				if len(events) < MAX_EVENTS_REQ {
					done = true
				}
				if c.rate_limit_set {
					time.Sleep(time.Duration(60/c.rpm) * time.Second)
				} else if c.throttle {
					time.Sleep(time.Duration(c.sleep) * time.Second)
					c.throttle = false
				}
			} else {
				err_out = e
				logErr(err_out)
				done = true
			}
			i++
		}

	}
	return
}

/*
 * V3 Group by urlname
 *
 * GET api.meetup.com/:urlname
 * Input:
 *	urlname of group
 *
 * Output:
 *	JsonResponse.
 */
func (c *MeetupClient) GetGroupByUrlname(url_name string) (group JsonResponse, err_out error)  {

	if len(url_name) == 0 {
		err_out = errors.New("Invalid request.")
		return
	}

	if req_url, e := url.Parse(API_BASE + "/" + url_name); e != nil {
		log.Println("Error parsing group urlname provided.")
		err_out = e
	} else {
		group, err_out = c.doV3GetObj(req_url)
	}
	return
}

