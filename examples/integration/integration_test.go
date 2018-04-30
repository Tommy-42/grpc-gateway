package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	gw "github.com/grpc-ecosystem/grpc-gateway/examples/proto/examplepb"
	"github.com/grpc-ecosystem/grpc-gateway/examples/proto/pathenum"
	"github.com/grpc-ecosystem/grpc-gateway/examples/proto/sub"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
)

type errorBody struct {
	Error   string        `json:"error"`
	Code    int           `json:"code"`
	Details []interface{} `json:"details"`
}

// TestEcho tests the most fundamental functionalities of marshaling requests
// an unmarshaling responses.
func TestEcho(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	testEcho(t, 8080, "application/json")
	testEchoOneof(t, 8080, "application/json")
	testEchoOneof1(t, 8080, "application/json")
	testEchoOneof2(t, 8080, "application/json")
	testEchoBody(t, 8080)
}

func TestForwardResponseOption(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		if err := runGateway(
			ctx,
			":8081",
			runtime.WithForwardResponseOption(
				func(_ context.Context, w http.ResponseWriter, _ proto.Message) error {
					w.Header().Set("Content-Type", "application/vnd.docker.plugins.v1.1+json")
					return nil
				},
			),
		); err != nil {
			t.Errorf("runGateway() failed with %v; want success", err)
			return
		}
	}()
	if err := waitForGateway(ctx, 8081); err != nil {
		t.Errorf("waitForGateway(ctx, 8081) failed with %v; want success", err)
	}
	testEcho(t, 8081, "application/vnd.docker.plugins.v1.1+json")
}

// testEcho is a part of TestEcho which deals with path parameters
func testEcho(t *testing.T, port int, contentType string) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/echo/myid", port)
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.SimpleMessage
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got, want := msg.Id, "myid"; got != want {
		t.Errorf("msg.Id = %q; want %q", got, want)
	}

	if value := resp.Header.Get("Content-Type"); value != contentType {
		t.Errorf("Content-Type was %s, wanted %s", value, contentType)
	}
}

func testEchoOneof(t *testing.T, port int, contentType string) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/echo/myid/10/golang", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.SimpleMessage
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got, want := msg.GetLang(), "golang"; got != want {
		t.Errorf("msg.GetLang() = %q; want %q", got, want)
	}

	if value := resp.Header.Get("Content-Type"); value != contentType {
		t.Errorf("Content-Type was %s, wanted %s", value, contentType)
	}
}

func testEchoOneof1(t *testing.T, port int, contentType string) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/echo1/myid/10/golang", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.SimpleMessage
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got, want := msg.GetStatus().GetNote(), "golang"; got != want {
		t.Errorf("msg.GetStatus().GetNote() = %q; want %q", got, want)
	}

	if value := resp.Header.Get("Content-Type"); value != contentType {
		t.Errorf("Content-Type was %s, wanted %s", value, contentType)
	}
}

func testEchoOneof2(t *testing.T, port int, contentType string) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/echo2/golang", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.SimpleMessage
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got, want := msg.GetNo().GetNote(), "golang"; got != want {
		t.Errorf("msg.GetNo().GetNote() = %q; want %q", got, want)
	}

	if value := resp.Header.Get("Content-Type"); value != contentType {
		t.Errorf("Content-Type was %s, wanted %s", value, contentType)
	}
}

// testEchoBody is a part of TestEcho which deals with a body parameter.
func testEchoBody(t *testing.T, port int) {
	sent := gw.SimpleMessage{Id: "example"}
	var m jsonpb.Marshaler
	payload, err := m.MarshalToString(&sent)
	if err != nil {
		t.Fatalf("m.MarshalToString(%#v) failed with %v; want success", payload, err)
	}

	url := fmt.Sprintf("http://localhost:%d/v1/example/echo_body", port)
	resp, err := http.Post(url, "", strings.NewReader(payload))
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var received gw.SimpleMessage
	if err := jsonpb.UnmarshalString(string(buf), &received); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got, want := received, sent; !reflect.DeepEqual(got, want) {
		t.Errorf("msg.Id = %q; want %q", got, want)
	}

	if got, want := resp.Header.Get("Grpc-Metadata-Foo"), "foo1"; got != want {
		t.Errorf("Grpc-Metadata-Foo was %q, wanted %q", got, want)
	}
	if got, want := resp.Header.Get("Grpc-Metadata-Bar"), "bar1"; got != want {
		t.Errorf("Grpc-Metadata-Bar was %q, wanted %q", got, want)
	}

	if got, want := resp.Trailer.Get("Grpc-Trailer-Foo"), "foo2"; got != want {
		t.Errorf("Grpc-Trailer-Foo was %q, wanted %q", got, want)
	}
	if got, want := resp.Trailer.Get("Grpc-Trailer-Bar"), "bar2"; got != want {
		t.Errorf("Grpc-Trailer-Bar was %q, wanted %q", got, want)
	}
}

// TestABE covers more than TestEcho.
// It covers (1) various HTTP methods; (2) various HTTP status codes
// (3) streaming requests and responses; (4) various types of fields;
// (5) various depths of nested fields; and (6) even more features like
// additional_bindings
func TestABE(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	testABECreate(t, 8080)
	testABECreateBody(t, 8080)
	testABEBulkCreate(t, 8080)
	testABELookup(t, 8080)
	testABELookupNotFound(t, 8080)
	testABEList(t, 8080)
	testABEBulkEcho(t, 8080)
	testABEBulkEchoZeroLength(t, 8080)
	testAdditionalBindings(t, 8080)
	testABERepeated(t, 8080)
}

// testABECreate tests path parameters with various types of fields.
func testABECreate(t *testing.T, port int) {
	const oneofStr = "bar"
	want := gw.ABitOfEverything{
		FloatValue:               1.5,
		DoubleValue:              2.5,
		Int64Value:               4294967296,
		Uint64Value:              9223372036854775807,
		Int32Value:               -2147483648,
		Fixed64Value:             9223372036854775807,
		Fixed32Value:             4294967295,
		BoolValue:                true,
		StringValue:              "strprefix/foo",
		Uint32Value:              4294967295,
		Sfixed32Value:            2147483647,
		Sfixed64Value:            -4611686018427387904,
		Sint32Value:              2147483647,
		Sint64Value:              4611686018427387903,
		OneofValue:               &gw.ABitOfEverything_OneofValueString{oneofStr},
		NonConventionalNameValue: "camelCase",
		EnumValue:                gw.NumericEnum_ZERO,
		PathEnumValue:            pathenum.PathEnum_DEF,
		NestedPathEnumValue:      pathenum.MessagePathEnum_JKL,
	}
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything/%f/%f/%d/separator/%d/%d/%d/%d/%v/%s/%d/%d/%d/%d/%d/%s/%s/%s/%s/%s", port, want.FloatValue, want.DoubleValue, want.Int64Value, want.Uint64Value, want.Int32Value, want.Fixed64Value, want.Fixed32Value, want.BoolValue, want.StringValue, want.Uint32Value, want.Sfixed32Value, want.Sfixed64Value, want.Sint32Value, want.Sint64Value, oneofStr, want.NonConventionalNameValue, want.EnumValue, want.PathEnumValue, want.NestedPathEnumValue)

	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.ABitOfEverything
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if msg.Uuid == "" {
		t.Error("msg.Uuid is empty; want not empty")
	}
	msg.Uuid = ""
	if got := msg; !reflect.DeepEqual(got, want) {
		t.Errorf("msg= %v; want %v", &got, &want)
	}
}

// testABECreateBody tests body parameters with various types of fields.
func testABECreateBody(t *testing.T, port int) {
	want := gw.ABitOfEverything{
		FloatValue:               1.5,
		DoubleValue:              2.5,
		Int64Value:               4294967296,
		Uint64Value:              9223372036854775807,
		Int32Value:               -2147483648,
		Fixed64Value:             9223372036854775807,
		Fixed32Value:             4294967295,
		BoolValue:                true,
		StringValue:              "strprefix/foo",
		Uint32Value:              4294967295,
		Sfixed32Value:            2147483647,
		Sfixed64Value:            -4611686018427387904,
		Sint32Value:              2147483647,
		Sint64Value:              4611686018427387903,
		NonConventionalNameValue: "camelCase",
		EnumValue:                gw.NumericEnum_ONE,
		PathEnumValue:            pathenum.PathEnum_ABC,
		NestedPathEnumValue:      pathenum.MessagePathEnum_GHI,

		Nested: []*gw.ABitOfEverything_Nested{
			{
				Name:   "bar",
				Amount: 10,
			},
			{
				Name:   "baz",
				Amount: 20,
			},
		},
		RepeatedStringValue: []string{"a", "b", "c"},
		OneofValue: &gw.ABitOfEverything_OneofValueString{
			OneofValueString: "x",
		},
		MapValue: map[string]gw.NumericEnum{
			"a": gw.NumericEnum_ONE,
			"b": gw.NumericEnum_ZERO,
		},
		MappedStringValue: map[string]string{
			"a": "x",
			"b": "y",
		},
		MappedNestedValue: map[string]*gw.ABitOfEverything_Nested{
			"a": {Name: "x", Amount: 1},
			"b": {Name: "y", Amount: 2},
		},
	}
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything", port)
	var m jsonpb.Marshaler
	payload, err := m.MarshalToString(&want)
	if err != nil {
		t.Fatalf("m.MarshalToString(%#v) failed with %v; want success", want, err)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.ABitOfEverything
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if msg.Uuid == "" {
		t.Error("msg.Uuid is empty; want not empty")
	}
	msg.Uuid = ""
	if got := msg; !reflect.DeepEqual(got, want) {
		t.Errorf("msg= %v; want %v", &got, &want)
	}
}

// testABEBulkCreate tests client-streaming
func testABEBulkCreate(t *testing.T, port int) {
	count := 0
	r, w := io.Pipe()
	go func(w io.WriteCloser) {
		defer func() {
			if cerr := w.Close(); cerr != nil {
				t.Errorf("w.Close() failed with %v; want success", cerr)
			}
		}()
		for _, val := range []string{
			"foo", "bar", "baz", "qux", "quux",
		} {
			want := gw.ABitOfEverything{
				FloatValue:               1.5,
				DoubleValue:              2.5,
				Int64Value:               4294967296,
				Uint64Value:              9223372036854775807,
				Int32Value:               -2147483648,
				Fixed64Value:             9223372036854775807,
				Fixed32Value:             4294967295,
				BoolValue:                true,
				StringValue:              fmt.Sprintf("strprefix/%s", val),
				Uint32Value:              4294967295,
				Sfixed32Value:            2147483647,
				Sfixed64Value:            -4611686018427387904,
				Sint32Value:              2147483647,
				Sint64Value:              4611686018427387903,
				NonConventionalNameValue: "camelCase",
				EnumValue:                gw.NumericEnum_ONE,
				PathEnumValue:            pathenum.PathEnum_ABC,
				NestedPathEnumValue:      pathenum.MessagePathEnum_GHI,

				Nested: []*gw.ABitOfEverything_Nested{
					{
						Name:   "hoge",
						Amount: 10,
					},
					{
						Name:   "fuga",
						Amount: 20,
					},
				},
			}
			var m jsonpb.Marshaler
			if err := m.Marshal(w, &want); err != nil {
				t.Fatalf("m.Marshal(%#v, w) failed with %v; want success", want, err)
			}
			if _, err := io.WriteString(w, "\n"); err != nil {
				t.Errorf("w.Write(%q) failed with %v; want success", "\n", err)
				return
			}
			count++
		}
	}(w)
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything/bulk", port)
	resp, err := http.Post(url, "application/json", r)
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg empty.Empty
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}

	if got, want := resp.Header.Get("Grpc-Metadata-Count"), fmt.Sprintf("%d", count); got != want {
		t.Errorf("Grpc-Metadata-Count was %q, wanted %q", got, want)
	}

	if got, want := resp.Trailer.Get("Grpc-Trailer-Foo"), "foo2"; got != want {
		t.Errorf("Grpc-Trailer-Foo was %q, wanted %q", got, want)
	}
	if got, want := resp.Trailer.Get("Grpc-Trailer-Bar"), "bar2"; got != want {
		t.Errorf("Grpc-Trailer-Bar was %q, wanted %q", got, want)
	}
}

// testABELookup tests unary call
func testABELookup(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything", port)
	cresp, err := http.Post(url, "application/json", strings.NewReader(`
		{"bool_value": true, "string_value": "strprefix/example"}
	`))
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer cresp.Body.Close()
	buf, err := ioutil.ReadAll(cresp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(cresp.Body) failed with %v; want success", err)
		return
	}
	if got, want := cresp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
		return
	}

	var want gw.ABitOfEverything
	if err := jsonpb.UnmarshalString(string(buf), &want); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &want) failed with %v; want success", buf, err)
		return
	}

	url = fmt.Sprintf("%s/%s", url, want.Uuid)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	var msg gw.ABitOfEverything
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got := msg; !reflect.DeepEqual(got, want) {
		t.Errorf("msg= %v; want %v", &got, &want)
	}

	if got, want := resp.Header.Get("Grpc-Metadata-Uuid"), want.Uuid; got != want {
		t.Errorf("Grpc-Metadata-Uuid was %s, wanted %s", got, want)
	}
}

// testABELookupNotFound tests HTTP status codes other than StatusOK.
// More coverage on status codes should be done in runtime/error_test.go.
func testABELookupNotFound(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything", port)
	uuid := "not_exist"
	url = fmt.Sprintf("%s/%s", url, uuid)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
		return
	}

	var msg errorBody
	if err := json.Unmarshal(buf, &msg); err != nil {
		t.Errorf("json.Unmarshal(%s, &msg) failed with %v; want success", buf, err)
		return
	}

	if got, want := msg.Code, int(codes.NotFound); got != want {
		t.Errorf("msg.Code = %d; want %d", got, want)
		return
	}

	if got, want := msg.Error, "not found"; got != want {
		t.Errorf("msg.Error = %s; want %s", got, want)
		return
	}

	if got, want := resp.Header.Get("Grpc-Metadata-Uuid"), uuid; got != want {
		t.Errorf("Grpc-Metadata-Uuid was %s, wanted %s", got, want)
	}
	if got, want := resp.Trailer.Get("Grpc-Trailer-Foo"), "foo2"; got != want {
		t.Errorf("Grpc-Trailer-Foo was %q, wanted %q", got, want)
	}
	if got, want := resp.Trailer.Get("Grpc-Trailer-Bar"), "bar2"; got != want {
		t.Errorf("Grpc-Trailer-Bar was %q, wanted %q", got, want)
	}
}

// testABEList tests server-streaming.
func testABEList(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var i int
	for i = 0; ; i++ {
		var item struct {
			Result json.RawMessage        `json:"result"`
			Error  map[string]interface{} `json:"error"`
		}
		err := dec.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("dec.Decode(&item) failed with %v; want success; i = %d", err, i)
		}
		if len(item.Error) != 0 {
			t.Errorf("item.Error = %#v; want empty; i = %d", item.Error, i)
			continue
		}
		var msg gw.ABitOfEverything
		if err := jsonpb.UnmarshalString(string(item.Result), &msg); err != nil {
			t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", item.Result, err)
		}
	}
	if i <= 0 {
		t.Errorf("i == %d; want > 0", i)
	}

	value := resp.Header.Get("Grpc-Metadata-Count")
	if value == "" {
		t.Errorf("Grpc-Metadata-Count should not be empty")
	}

	count, err := strconv.Atoi(value)
	if err != nil {
		t.Errorf("failed to Atoi %q: %v", value, err)
	}

	if count <= 0 {
		t.Errorf("count == %d; want > 0", count)
	}
}

// testABEBulkEcho tests bidi-streaming
func testABEBulkEcho(t *testing.T, port int) {
	reqr, reqw := io.Pipe()
	var wg sync.WaitGroup
	var want []*sub.StringMessage
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer reqw.Close()
		var m jsonpb.Marshaler
		for i := 0; i < 1000; i++ {
			msg := sub.StringMessage{Value: proto.String(fmt.Sprintf("message %d", i))}
			buf, err := m.MarshalToString(&msg)
			if err != nil {
				t.Errorf("m.Marshal(%v) failed with %v; want success", &msg, err)
				return
			}
			if _, err := fmt.Fprintln(reqw, buf); err != nil {
				t.Errorf("fmt.Fprintln(reqw, %q) failed with %v; want success", buf, err)
				return
			}
			want = append(want, &msg)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything/echo", port)
	req, err := http.NewRequest("POST", url, reqr)
	if err != nil {
		t.Errorf("http.NewRequest(%q, %q, reqr) failed with %v; want success", "POST", url, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("http.Post(%q, %q, req) failed with %v; want success", url, "application/json", err)
		return
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
	}

	var got []*sub.StringMessage
	wg.Add(1)
	go func() {
		defer wg.Done()

		dec := json.NewDecoder(resp.Body)
		for i := 0; ; i++ {
			var item struct {
				Result json.RawMessage        `json:"result"`
				Error  map[string]interface{} `json:"error"`
			}
			err := dec.Decode(&item)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("dec.Decode(&item) failed with %v; want success; i = %d", err, i)
			}
			if len(item.Error) != 0 {
				t.Errorf("item.Error = %#v; want empty; i = %d", item.Error, i)
				continue
			}
			var msg sub.StringMessage
			if err := jsonpb.UnmarshalString(string(item.Result), &msg); err != nil {
				t.Errorf("jsonpb.UnmarshalString(%q, &msg) failed with %v; want success", item.Result, err)
			}
			got = append(got, &msg)
		}
	}()

	wg.Wait()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got = %v; want %v", got, want)
	}
}

// testABEBulkEchoZeroLength covers an edge case of testABEBulkEcho that the length of
// the chunked stream is zero.
func testABEBulkEchoZeroLength(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything/echo", port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(nil))
	if err != nil {
		t.Errorf("http.NewRequest(%q, %q, bytes.NewReader(nil)) failed with %v; want success", "POST", url, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("http.Post(%q, %q, req) failed with %v; want success", url, "application/json", err)
		return
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
	}

	dec := json.NewDecoder(resp.Body)
	var item struct {
		Result json.RawMessage        `json:"result"`
		Error  map[string]interface{} `json:"error"`
	}
	if err := dec.Decode(&item); err == nil {
		t.Errorf("dec.Decode(&item) succeeded; want io.EOF; item = %#v", item)
	} else if err != io.EOF {
		t.Errorf("dec.Decode(&item) failed with %v; want success", err)
		return
	}
}

// testAdditionalBindings tests if additional_bindings works as fine as the
// primary binding.
func testAdditionalBindings(t *testing.T, port int) {
	for i, f := range []func() *http.Response{
		func() *http.Response {
			url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything/echo/hello", port)
			resp, err := http.Get(url)
			if err != nil {
				t.Errorf("http.Get(%q) failed with %v; want success", url, err)
				return nil
			}
			return resp
		},
		func() *http.Response {
			url := fmt.Sprintf("http://localhost:%d/v2/example/echo", port)
			resp, err := http.Post(url, "application/json", strings.NewReader(`"hello"`))
			if err != nil {
				t.Errorf("http.Post(%q, %q, %q) failed with %v; want success", url, "application/json", `"hello"`, err)
				return nil
			}
			return resp
		},
		func() *http.Response {
			r, w := io.Pipe()
			go func() {
				defer w.Close()
				w.Write([]byte(`"hello"`))
			}()
			url := fmt.Sprintf("http://localhost:%d/v2/example/echo", port)
			resp, err := http.Post(url, "application/json", r)
			if err != nil {
				t.Errorf("http.Post(%q, %q, %q) failed with %v; want success", url, "application/json", `"hello"`, err)
				return nil
			}
			return resp
		},
		func() *http.Response {
			url := fmt.Sprintf("http://localhost:%d/v2/example/echo?value=hello", port)
			resp, err := http.Get(url)
			if err != nil {
				t.Errorf("http.Get(%q) failed with %v; want success", url, err)
				return nil
			}
			return resp
		},
	} {
		resp := f()
		if resp == nil {
			continue
		}

		defer resp.Body.Close()
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success; i=%d", err, i)
			return
		}
		if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Errorf("resp.StatusCode = %d; want %d; i=%d", got, want, i)
			t.Logf("%s", buf)
		}

		var msg sub.StringMessage
		if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
			t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success; %d", buf, err, i)
			return
		}
		if got, want := msg.GetValue(), "hello"; got != want {
			t.Errorf("msg.GetValue() = %q; want %q", got, want)
		}
	}
}

func testABERepeated(t *testing.T, port int) {
	f := func(v reflect.Value) string {
		var f func(v reflect.Value, idx int) string
		s := make([]string, v.Len())
		switch v.Index(0).Kind() {
		case reflect.Slice:
			f = func(v reflect.Value, idx int) string {
				t := v.Index(idx).Type().Elem().Kind()
				if t == reflect.Uint8 {
					return base64.URLEncoding.EncodeToString(v.Index(idx).Interface().([]byte))
				}
				// Could handle more elegantly
				panic("unknown slice of type: " + t.String())
			}
		default:
			f = func(v reflect.Value, idx int) string {
				return fmt.Sprintf("%v", v.Index(idx).Interface())
			}
		}
		for i := 0; i < v.Len(); i++ {
			s[i] = f(v, i)
		}
		return strings.Join(s, ",")
	}
	want := gw.ABitOfEverythingRepeated{
		PathRepeatedFloatValue: []float32{
			1.5,
			-1.5,
		},
		PathRepeatedDoubleValue: []float64{
			2.5,
			-2.5,
		},
		PathRepeatedInt64Value: []int64{
			4294967296,
			-4294967296,
		},
		PathRepeatedUint64Value: []uint64{
			0,
			9223372036854775807,
		},
		PathRepeatedInt32Value: []int32{
			2147483647,
			-2147483648,
		},
		PathRepeatedFixed64Value: []uint64{
			0,
			9223372036854775807,
		},
		PathRepeatedFixed32Value: []uint32{
			0,
			4294967295,
		},
		PathRepeatedBoolValue: []bool{
			true,
			false,
		},
		PathRepeatedStringValue: []string{
			"foo",
			"bar",
		},
		PathRepeatedBytesValue: [][]byte{
			[]byte{0x00},
			[]byte{0xFF},
		},
		PathRepeatedUint32Value: []uint32{
			0,
			4294967295,
		},
		PathRepeatedEnumValue: []gw.NumericEnum{
			gw.NumericEnum_ZERO,
			gw.NumericEnum_ONE,
		},
		PathRepeatedSfixed32Value: []int32{
			2147483647,
			-2147483648,
		},
		PathRepeatedSfixed64Value: []int64{
			4294967296,
			-4294967296,
		},
		PathRepeatedSint32Value: []int32{
			2147483647,
			-2147483648,
		},
		PathRepeatedSint64Value: []int64{
			4611686018427387903,
			-4611686018427387904,
		},
	}
	url := fmt.Sprintf("http://localhost:%d/v1/example/a_bit_of_everything_repeated/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s", port, f(reflect.ValueOf(want.PathRepeatedFloatValue)), f(reflect.ValueOf(want.PathRepeatedDoubleValue)), f(reflect.ValueOf(want.PathRepeatedInt64Value)), f(reflect.ValueOf(want.PathRepeatedUint64Value)), f(reflect.ValueOf(want.PathRepeatedInt32Value)), f(reflect.ValueOf(want.PathRepeatedFixed64Value)), f(reflect.ValueOf(want.PathRepeatedFixed32Value)), f(reflect.ValueOf(want.PathRepeatedBoolValue)), f(reflect.ValueOf(want.PathRepeatedStringValue)), f(reflect.ValueOf(want.PathRepeatedBytesValue)), f(reflect.ValueOf(want.PathRepeatedUint32Value)), f(reflect.ValueOf(want.PathRepeatedEnumValue)), f(reflect.ValueOf(want.PathRepeatedSfixed32Value)), f(reflect.ValueOf(want.PathRepeatedSfixed64Value)), f(reflect.ValueOf(want.PathRepeatedSint32Value)), f(reflect.ValueOf(want.PathRepeatedSint64Value)))

	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	var msg gw.ABitOfEverythingRepeated
	if err := jsonpb.UnmarshalString(string(buf), &msg); err != nil {
		t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success", buf, err)
		return
	}
	if got := msg; !reflect.DeepEqual(got, want) {
		t.Errorf("msg= %v; want %v", &got, &want)
	}
}

// TestDeepFields tests deeply nested fields in path parameters.
//
// NOTE: deeply nested fields are not allowed in body parameters.
// c.f. third_party/googleapis/google/api/http.proto
// > the referred field must not be a repeated field and must be
// > present at the top-level of request message type.
func TestDeepFields(t *testing.T) {
	const base = "http://localhost:8080"

	for i, spec := range []struct {
		path    string
		payload string
		want    gw.ABitOfEverything
	}{
		{
			path: "/v1/example/a_bit_of_everything/foo",
			want: gw.ABitOfEverything{
				SingleNested: &gw.ABitOfEverything_Nested{
					Name: "foo",
				},
			},
		},
		{
			path: "/v2/example/single_nested/deeper_nested_value/foo",
			want: gw.ABitOfEverything{
				SingleNested: &gw.ABitOfEverything_Nested{
					DeeperNestedValue: &sub.StringMessage{
						Value: proto.String("foo"),
					},
				},
			},
		},
	} {
		url := base + spec.path
		resp, err := http.Post(url, "application/json", strings.NewReader(spec.payload))
		if err != nil {
			t.Errorf("http.Post(%q) failed with %v; want success; at i=%d", url, err, i)
			return
		}
		defer resp.Body.Close()
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("iotuil.ReadAll(resp.Body) failed with %v; want success; at i=%d", err, i)
			return
		}

		if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Errorf("resp.StatusCode = %d; want %d; at i=%d", got, want, i)
			t.Logf("%s", buf)
		}

		var got gw.ABitOfEverything
		if err := jsonpb.UnmarshalString(string(buf), &got); err != nil {
			t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success; at i=%d", buf, err, i)
			return
		}
		if !reflect.DeepEqual(got, spec.want) {
			t.Errorf("got = %v; want %v; at i=%d", got, spec.want, i)
		}
	}
}

func TestOneof(t *testing.T) {
	const base = "http://localhost:8080"

	for i, spec := range []struct {
		path    string
		payload string
		want    gw.ABitOfEverything
	}{
		// path param: primitive field in a oneof clause.
		{
			path: "/v2/example/oneof_value_string/echo/foo",
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueString{"foo"},
			},
		},
		// path param: primitive field in a message field in a oneof clause
		{
			path: "/v2/example/oneof_value_nested/name/echo/foo",
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueNested{
					&gw.ABitOfEverything_Nested{
						Name: "foo",
					},
				},
			},
		},
		// path param: primitive field in a message field in a message field in a oneof clause
		{
			path: "/v2/example/oneof_value_nested/deeper_nested_value/value/echo/foo",
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueNested{
					&gw.ABitOfEverything_Nested{
						DeeperNestedValue: &sub.StringMessage{
							Value: proto.String("foo"),
						},
					},
				},
			},
		},
		// path param: primitive field in a oneof clause in a message field in a oneof clause.
		{
			path: "/v2/example/oneof_value_nested/oneof_value_terminal/echo/foo",
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueNested{
					&gw.ABitOfEverything_Nested{
						OneofValue: &gw.ABitOfEverything_Nested_OneofValueTerminal{
							"foo",
						},
					},
				},
			},
		},
		// path param: primitive field in a message field in a oneof clause in a message field in a oneof clause.
		{
			path: "/v2/example/oneof_value_nested/oneof_value_deeper_nested/value/echo/foo",
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueNested{
					&gw.ABitOfEverything_Nested{
						OneofValue: &gw.ABitOfEverything_Nested_OneofValueDeeperNested{
							&sub.StringMessage{
								Value: proto.String("foo"),
							},
						},
					},
				},
			},
		},

		// body param: primitive field in a oneof clause
		{
			path:    "/v2/example/oneof_value_string/echo",
			payload: `"foo"`,
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueString{"foo"},
			},
		},
		// body param: message field in a oneof clause
		{
			path: "/v2/example/oneof_value_nested/echo",
			payload: `
					{
						"name": "foo",
						"amount": 123
					}
				`,
			want: gw.ABitOfEverything{
				OneofValue: &gw.ABitOfEverything_OneofValueNested{
					&gw.ABitOfEverything_Nested{
						Name:   "foo",
						Amount: 123,
					},
				},
			},
		},
		// NOTE: deeply nested fields are not allowed in body by spec.

	} {
		url := base + spec.path
		resp, err := http.Post(url, "application/json", strings.NewReader(spec.payload))
		if err != nil {
			t.Errorf("http.Post(%q) failed with %v; want success; at i=%d", url, err, i)
			return
		}
		defer resp.Body.Close()
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("iotuil.ReadAll(resp.Body) failed with %v; want success; at i=%d", err, i)
			return
		}

		if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Errorf("resp.StatusCode = %d; want %d; at i=%d", got, want, i)
			t.Logf("%s", buf)
		}

		var got gw.ABitOfEverything
		if err := jsonpb.UnmarshalString(string(buf), &got); err != nil {
			t.Errorf("jsonpb.UnmarshalString(%s, &msg) failed with %v; want success; at i=%d", buf, err, i)
			return
		}
		if !reflect.DeepEqual(got, spec.want) {
			t.Errorf("got = %v; want %v; at i=%d", &got, &spec.want, i)
		}
	}
}

// TestTimeout tests if timeout metadata propagates to the grpc backend server.
// TODO(yugui) Check if the timeout is actually earlier than the default HTTP timeout.
func TestTimeout(t *testing.T) {
	url := "http://localhost:8080/v2/example/timeout"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf(`http.NewRequest("GET", %q, nil) failed with %v; want success`, url, err)
		return
	}
	req.Header.Set("Grpc-Timeout", "10m")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("http.DefaultClient.Do(%#v) failed with %v; want success", req, err)
		return
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusGatewayTimeout; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
	}
}

// TestErrorWithDetails tests error details set by grpc server are available
// in the HTTP response.
func TestErrorWithDetails(t *testing.T) {
	url := "http://localhost:8080/v2/example/errorwithdetails"
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
	}

	if got, want := resp.StatusCode, http.StatusInternalServerError; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
	}

	var msg errorBody
	if err := json.Unmarshal(buf, &msg); err != nil {
		t.Fatalf("json.Unmarshal(%s, &msg) failed with %v; want success", buf, err)
	}

	if got, want := msg.Code, int(codes.Unknown); got != want {
		t.Errorf("msg.Code = %d; want %d", got, want)
	}
	if got, want := msg.Error, "with details"; got != want {
		t.Errorf("msg.Error = %s; want %s", got, want)
	}
	if got, want := len(msg.Details), 1; got != want {
		t.Fatalf("len(msg.Details) = %q; want %q", got, want)
	}

	details, ok := msg.Details[0].(map[string]interface{})
	if got, want := ok, true; got != want {
		t.Fatalf("msg.Details[0] got type: %T, want %T", msg.Details[0], map[string]interface{}{})
	}
	typ, ok := details["@type"].(string)
	if got, want := ok, true; got != want {
		t.Fatalf("msg.Details[0][\"@type\"] got type: %T, want %T", typ, "")
	}
	if got, want := details["@type"], "type.googleapis.com/google.rpc.DebugInfo"; got != want {
		t.Errorf("msg.Details[\"@type\"] = %q; want %q", got, want)
	}
	if got, want := details["detail"], "error debug details"; got != want {
		t.Errorf("msg.Details[\"detail\"] = %q; want %q", got, want)
	}
	entries, ok := details["stack_entries"].([]interface{})
	if got, want := ok, true; got != want {
		t.Fatalf("msg.Details[0][\"stack_entries\"] got type: %T, want %T", entries, []string{})
	}
	entry, ok := entries[0].(string)
	if got, want := ok, true; got != want {
		t.Fatalf("msg.Details[0][\"stack_entries\"][0] got type: %T, want %T", entry, "")
	}
	if got, want := entries[0], "foo:1"; got != want {
		t.Errorf("msg.Details[\"stack_entries\"][0] = %q; want %q", got, want)
	}
}

// TestPostWithEmptyBody covers an edge-case that the request body is empty.
// It should be valid, in particular, all fields are specified in path
// parameters and query parameters.
func TestPostWithEmptyBody(t *testing.T) {
	url := "http://localhost:8080/v2/example/postwithemptybody/name"
	rep, err := http.Post(url, "application/json", nil)

	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}

	if rep.StatusCode != http.StatusOK {
		t.Errorf("http.Post(%q) response code is %d; want %d", url,
			rep.StatusCode, http.StatusOK)
		return
	}
}

// TestUnknownPath tests if the router rejects unknown paths.
func TestUnknownPath(t *testing.T) {
	url := "http://localhost:8080"
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}
}

// TestUnknownPath tests if the router rejects unsupported methods on a
// known path.
func TestMethodNotAllowed(t *testing.T) {
	url := "http://localhost:8080/v1/example/echo/myid"
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Post(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusMethodNotAllowed; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}
}

// TestInvalidArgument tests if the decoder rejects a malformed path parameter.
func TestInvalidArgument(t *testing.T) {
	url := "http://localhost:8080/v1/example/echo/myid/not_int64"
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}
}

func TestResponseBody(t *testing.T) {
	if testing.Short() {
		t.Skip()
		return
	}

	testResponseBody(t, 8080)
}

func testResponseBody(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/responsebody/foo", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("http.Get(%q) failed with %v; want success", url, err)
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll(resp.Body) failed with %v; want success", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d; want %d", got, want)
		t.Logf("%s", buf)
	}

	if got, want := string(buf), `{"data":"foo"}`; got != want {
		t.Errorf("response = %q; want %q", got, want)
	}
}
