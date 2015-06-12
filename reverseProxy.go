package main

import(
	"container/ring"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"net/http/httptest"
	"io/ioutil"
	"bytes"
	"fmt"
)

func startServer() {
	//start port
	sourceAddress := ":8080"
	destinationSite := "http://127.0.0.1"
	//destination site
	ports:= []string{
		":3000",
		":4000",
	}

	//using ring to take balance
	hostRing := ring.New(len(ports))
	for _, port := range ports {
		url, _ := url.Parse(destinationSite +port)
		hostRing.Value = url
		hostRing = hostRing.Next()
	}

	mutex := sync.Mutex{}
	director := func(request *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		request.URL.Scheme = "http"
		request.URL.Host = hostRing.Value.(*url.URL).Host
		hostRing = hostRing.Next()
	}

	//create reverse proxy
	proxy := &httputil.ReverseProxy{Director: director}

	//transport to customise roundtrip
	proxy.Transport = &myTransport{}
	server := http.Server{
		Addr: sourceAddress,
		Handler: proxy,
	}
	server.ListenAndServe()
}

//transport
type myTransport struct {
	Transport http.RoundTripper
	DirPath string
	Num int32
}

//save response
type ResponseFixture struct {
	URL *url.URL
	Response *httptest.ResponseRecorder
}

var RDList [][]byte = make([][]byte, 0)

//rewrite request and response
func (t *myTransport) RoundTrip(r *http.Request) (*http.Response, error) {

	//judge whether illegle
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return &http.Response{}, err
	}

	//whether it is in cache or not
	for _,rf := range RDList{
		responseFixture := ResponseFixture{}
		responseFixture.UnmarshalJSON(rf)

		//cache contains it
		if ok := responseFixture.URL.String() == r.URL.String(); ok{
			resp := &http.Response{}
			resp.Body = ioutil.NopCloser(responseFixture.Response.Body)

			return resp, nil
		}
	}

	//hasn's cache
	re, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

    //parse body
	b := new(bytes.Buffer)
	b.ReadFrom(re.Body)
	rr := &httptest.ResponseRecorder{
		Code:re.StatusCode,
		HeaderMap:re.Header,
		Body:bytes.NewBuffer(b.Bytes()),
	}

	//save response
	responseFixture := &ResponseFixture{URL: u, Response: rr}
	re.Body = ioutil.NopCloser(b)
	//transfer to JSON
	j, err := responseFixture.MarshalJSON()
	if err != nil {
		fmt.Println(err)
	}

	RDList = append(RDList, j)

	//write to disk
	num := atomic.AddInt32(&t.Num, 1)
	path := filepath.Join(t.DirPath, "response"+strconv.Itoa(int(num-1))+".json")
	err = ioutil.WriteFile(path, j, 0644)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(len(RDList))
	return re, nil
}

//load data from Bytes and json
func (r *ResponseFixture) UnmarshalJSON(j []byte) error {
	obj := &jsonFixture{}
	err := json.Unmarshal(j, obj)
	if err != nil {
		return err
	}
	u, err := url.Parse(obj.URL)
	if err != nil {
		return err
	}
	re := &httptest.ResponseRecorder{
		Code:obj.Code,
		Body:bytes.NewBuffer([]byte(obj.Body)),
		HeaderMap:make(map[string][]string),
	}
	for k, v := range obj.HeaderMap {
		var arr []string
		copy(arr, v)
		re.HeaderMap[k] = arr
	}
	r.URL = u
	r.Response = re
	return nil
}

//transfer to JSON and Bytes
func (r *ResponseFixture) MarshalJSON() ([]byte, error) {
	//generate struct
	obj := &jsonFixture{
		URL:r.URL.String(),
		Code:r.Response.Code,
		Body:string(r.Response.Body.Bytes()),
		HeaderMap:make(map[string][]string),
	}
	//transfer to JSON
	for k, v := range r.Response.HeaderMap {
		obj.HeaderMap[k] = v
	}

	//ready to write to disk
	return json.Marshal(obj)
}

//JSON struct
type jsonFixture struct {
	URL string `json:"url"`
	Code int `json:"code"`
	Body string `json:"body"`
	HeaderMap map[string][]string `json:"headers"`
}

func config(){
//	RDList := make([]*jsonFixture, 10)
}

//main
func main(){
	config()
	startServer()
}
