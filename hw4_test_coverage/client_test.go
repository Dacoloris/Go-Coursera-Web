package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Row struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

const AccessToken = "1234"

func forRowToUser(r Row) User {
	return User{
		r.Id,
		r.FirstName + " " + r.LastName,
		r.Age,
		r.About,
		r.Gender,
	}
}

func (r Row) filter(q string) bool {
	name := r.FirstName + " " + r.LastName

	return strings.Contains(name, q) || strings.Contains(r.About, q)
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != AccessToken {
		http.Error(w, "Bad AccessToken", http.StatusUnauthorized)
		return
	}

	f, err := os.Open("dataset.xml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data Root
	err = xml.Unmarshal(b, &data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := r.URL.Query()

	query := params.Get("query")
	res := make([]User, 0)

	for _, row := range data.Rows {
		if row.filter(query) {
			res = append(res, forRowToUser(row))
		}
	}

	orderBy, _ := strconv.Atoi(params.Get("order_by"))
	err = sortUsers(res, params.Get("order_field"), orderBy)
	if err != nil {
		jsn, _ := json.Marshal(SearchErrorResponse{"ErrorBadOrderField"})
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, string(jsn))
		return
	}

	limit, _ := strconv.Atoi(params.Get("limit"))
	offset, _ := strconv.Atoi(params.Get("offset"))
	if offset >= len(res) {
		res = []User{}
	} else if limit+offset > len(res) {
		res = res[offset:len(res)]
	} else if limit > 0 {
		res = res[offset : offset+limit]
	}

	jsn, _ := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(jsn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sortUsers(users []User, orderField string, orderBy int) error {
	if orderBy == OrderByAsIs {
		return nil
	}

	var less func(u1, u2 User) bool

	switch orderField {
	case "Id":
		less = func(u1, u2 User) bool {
			return u1.Id < u2.Id
		}
	case "Name", "":
		less = func(u1, u2 User) bool {
			return u1.Name < u2.Name
		}
	case "Age":
		less = func(u1, u2 User) bool {
			return u1.Age < u2.Age
		}
	default:
		return fmt.Errorf(ErrorBadOrderField)
	}

	sort.Slice(users, func(i, j int) bool {
		return less(users[i], users[j]) && (orderBy == orderDesc)
	})

	return nil
}

type TestServer struct {
	server *httptest.Server
	client SearchClient
}

func newTestServer(token string) TestServer {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{token, server.URL}

	return TestServer{server, client}
}

func TestLimitNegative(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.server.Close()

	_, err := ts.client.FindUsers(SearchRequest{
		Limit: -1,
	})

	if err == nil {
		t.Errorf("Wrong answer")
	} else if err.Error() != "limit must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestOffsetNegative(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.server.Close()

	_, err := ts.client.FindUsers(SearchRequest{
		Offset: -1,
	})

	if err == nil {
		t.Errorf("Wrong answer")
	} else if err.Error() != "offset must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestAccessToken(t *testing.T) {
	ts := newTestServer("token")
	defer ts.server.Close()

	_, err := ts.client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Wrong answer")
	} else if err.Error() != "Bad AccessToken" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestInvalidOrderField(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.server.Close()

	_, err := ts.client.FindUsers(SearchRequest{
		OrderBy:    OrderByAsc,
		OrderField: "something",
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "OrderFeld something invalid" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestLimitHigh(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.server.Close()

	response, _ := ts.client.FindUsers(SearchRequest{
		Limit: 30,
	})

	if len(response.Users) != 25 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
	}
}

func TestFindUserByName(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.server.Close()

	response, _ := ts.client.FindUsers(SearchRequest{
		Query: "Hilda",
		Limit: 1,
	})

	if len(response.Users) != 1 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[0].Name != "Hilda Mayer" {
		t.Errorf("Invalid user found: %v", response.Users[0])
		return
	}
}

func TestLimitOffset(t *testing.T) {
	ts := newTestServer(AccessToken)
	defer ts.server.Close()

	response, _ := ts.client.FindUsers(SearchRequest{
		Limit:  5,
		Offset: 0,
	})

	if len(response.Users) != 5 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[2].Name != "Brooks Aguilar" {
		t.Errorf("Invalid user at position 3: %v", response.Users[2])
		return
	}

	response, _ = ts.client.FindUsers(SearchRequest{
		Limit:  5,
		Offset: 2,
	})

	if len(response.Users) != 5 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[0].Name != "Brooks Aguilar" {
		t.Errorf("Invalid user at position 3: %v", response.Users[0])
		return
	}
}

func TestFatalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Fatal Error", http.StatusInternalServerError)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "SearchServer fatal error" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCantUnpackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Some Error", http.StatusBadRequest)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownBadRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsn, _ := json.Marshal(SearchErrorResponse{"Unknown Error"})
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, string(jsn))
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCantUnpackResultError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "None")
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	client := SearchClient{AccessToken, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownError(t *testing.T) {
	client := SearchClient{AccessToken, "http://invalid-server/"}

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}
