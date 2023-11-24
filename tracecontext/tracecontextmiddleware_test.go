package tracecontext_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d-velop/dvelop-sdk-go/tracecontext"
)

func TestShouldCallInnerHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	innerHandler := handlerSpy{}

	tracecontext.AddToCtx()(&innerHandler).ServeHTTP(httptest.NewRecorder(), req)

	if !innerHandler.hasBeenCalled {
		t.Error("inner handler should have been called")
	}
}

func TestMissingTraceparentHeader_GeneratesNewTraceIdAndNewSpanId(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	innerHandler := handlerSpy{}

	tracecontext.AddToCtx()(&innerHandler).ServeHTTP(httptest.NewRecorder(), req)

	if err := innerHandler.assertTraceIdIsSet(); err != nil {
		t.Error(err)
	}
	if err := innerHandler.assertSpanIdIsSet(); err != nil {
		t.Error(err)
	}
}

func TestInvalidTraceparentHeader_GeneratesNewTraceIdAndNewSpanId(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	req.Header.Set("traceparent", "invalid")
	if err != nil {
		t.Fatal(err)
	}
	innerHandler := handlerSpy{}

	tracecontext.AddToCtx()(&innerHandler).ServeHTTP(httptest.NewRecorder(), req)

	if err := innerHandler.assertTraceIdIsSet(); err != nil {
		t.Error(err)
	}
	if err := innerHandler.assertSpanIdIsSet(); err != nil {
		t.Error(err)
	}
}

func TestTraceparentHeader_SetGivenTraceIdToCtxAndGeneratesNewSpanId(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if err != nil {
		t.Fatal(err)
	}
	innerHandler := handlerSpy{}

	tracecontext.AddToCtx()(&innerHandler).ServeHTTP(httptest.NewRecorder(), req)

	if err = innerHandler.assertTraceIdIs("4bf92f3577b34da6a3ce929d0e0e4736"); err != nil {
		t.Error(err)
	}
	if err = innerHandler.assertSpanIdIsSet(); err != nil {
		t.Error(err)
	}
	if err = innerHandler.assertSpanIdIsNot("00f067aa0ba902b7"); err != nil {
		t.Error(err)
	}
}

func TestTraceparentHeader_GetSameTraceparentWithNewSpanIdAndFlags1(t *testing.T) {
	req, err := http.NewRequest("GET", "/myresource/sub", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00")
	if err != nil {
		t.Fatal(err)
	}
	innerHandler := handlerSpy{}

	tracecontext.AddToCtx()(&innerHandler).ServeHTTP(httptest.NewRecorder(), req)

	if err := innerHandler.assertTraceparentIs(fmt.Sprintf("00-4bf92f3577b34da6a3ce929d0e0e4736-%v-01", innerHandler.spanId)); err != nil {
		t.Error(err)
	}
}

func TestNoTraceIdOnContext_WithTraceIdCtx_ReturnsContextWithTraceId(t *testing.T) {
	ctx := tracecontext.WithTraceIdCtx(context.Background(), "4bf92f3577b34da6a3ce929d0e0e4736")
	traceId, _ := tracecontext.TraceIdFromCtx(ctx)
	assertString(t, "4bf92f3577b34da6a3ce929d0e0e4736", traceId)
}

func TestNoSpanIdOnContext_WithSpanIdCtx_ReturnsContextWithSpanId(t *testing.T) {
	ctx := tracecontext.WithSpanIdCtx(context.Background(), "00f067aa0ba902b7")
	spanId, _ := tracecontext.SpanIdFromCtx(ctx)
	assertString(t, "00f067aa0ba902b7", spanId)
}

func TestGivenTraceIdAndSpanIdOnContext_TraceParentFromCtx_ReturnsTraceparentFromContext(t *testing.T) {
	ctx := tracecontext.WithSpanIdCtx(context.Background(), "00f067aa0ba902b7")
	ctx = tracecontext.WithTraceIdCtx(ctx, "4bf92f3577b34da6a3ce929d0e0e4736")
	traceparent, _ := tracecontext.TraceparentFromCtx(ctx)
	assertString(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", traceparent)
}

func TestRoundTripper_givenTraceparentInContext_thenAddsToHeader(t *testing.T) {
	ctx := tracecontext.WithSpanIdCtx(context.Background(), "00f067aa0ba902b7")
	ctx = tracecontext.WithTraceIdCtx(ctx, "4bf92f3577b34da6a3ce929d0e0e4736")
	req := httptest.NewRequest("GET", "/myresource/sub", nil).WithContext(ctx)
	mock := roundTripperMock{}

	resp, err := tracecontext.RoundTripper(&mock).RoundTrip(req)
	if err != nil {
		t.Errorf("err does not match\nExpected: %v\nGot:      %v", nil, err)
	}

	traceparentGot := resp.Request.Header.Get("traceparent")
	traceparentWant, _ := tracecontext.TraceparentFromCtx(ctx)
	if traceparentGot != traceparentWant {
		t.Errorf("traceparentGot does not match\nExpected: %s\nGot:      %s", traceparentWant, traceparentGot)
	}
}

func TestRoundTrip_givenNonTraceparentInContext_thenDoesNotAddToHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/myresource/sub", nil)
	mock := roundTripperMock{}

	resp, err := tracecontext.RoundTripper(&mock).RoundTrip(req)
	if err != nil {
		t.Errorf("err does not match\nExpected: %v\nGot:      %v", nil, err)
	}

	traceparentGot := resp.Request.Header.Get("traceparent")
	traceparentWant := ""
	if traceparentGot != traceparentWant {
		t.Errorf("traceparentGot does not match\nExpected: %s\nGot:      %s", traceparentWant, traceparentGot)
	}
}

type handlerSpy struct {
	hasBeenCalled bool
	traceparent   string
	traceId       string
	spanId        string
}

func (spy *handlerSpy) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	spy.hasBeenCalled = true
	spy.traceId, _ = tracecontext.TraceIdFromCtx(r.Context())
	spy.spanId, _ = tracecontext.SpanIdFromCtx(r.Context())
	spy.traceparent, _ = tracecontext.TraceparentFromCtx(r.Context())
}

func (spy *handlerSpy) assertTraceparentIs(expected string) error {
	if spy.traceparent != expected {
		return fmt.Errorf("handler set wrong traceparent on context: got %v want %v", spy.traceparent, expected)
	}
	return nil
}

func (spy *handlerSpy) assertTraceIdIs(expected string) error {
	if spy.traceId != expected {
		return fmt.Errorf("handler set wrong traceId on context: got %v want %v", spy.traceId, expected)
	}
	return nil
}

func (spy *handlerSpy) assertTraceIdIsSet() error {
	if spy.traceId == "" || len(spy.traceId) != 32 {
		return fmt.Errorf("handler did not set a traceId on context")
	}
	return nil
}

func (spy *handlerSpy) assertSpanIdIsNot(expected string) error {
	if spy.spanId == expected {
		return fmt.Errorf("handler set wrong spanId on context: got %v want %v", spy.spanId, expected)
	}
	return nil
}

func (spy *handlerSpy) assertSpanIdIsSet() error {
	if spy.spanId == "" || len(spy.spanId) != 16 {
		return fmt.Errorf("handler did not set a spanId on context")
	}
	return nil
}

type roundTripperMock struct {
}

func (rt *roundTripperMock) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{Request: req}, nil
}
