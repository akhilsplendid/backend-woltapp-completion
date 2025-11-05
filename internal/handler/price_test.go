package handler

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "backend-woltapp-completion/internal/homeapi"
)

// fake client for tests
type fakeClient struct{
    s homeapi.StaticData
    d homeapi.DynamicData
    sErr error
    dErr error
}
func (f *fakeClient) GetStatic(ctx context.Context, venueSlug string) (homeapi.StaticData, error) { return f.s, f.sErr }
func (f *fakeClient) GetDynamic(ctx context.Context, venueSlug string) (homeapi.DynamicData, error) { return f.d, f.dErr }

func TestHaversineMeters(t *testing.T) {
    // Distance from (0,0) to (0,1) ~ 111,319 meters
    d := haversineMeters(0, 0, 0, 1)
    if d < 111000 || d > 111500 {
        t.Fatalf("unexpected distance: %d", d)
    }
    // Same point should be zero
    if z := haversineMeters(10.5, 20.7, 10.5, 20.7); z != 0 {
        t.Fatalf("expected 0, got %d", z)
    }
}

func TestSelectRange(t *testing.T) {
    rs := []homeapi.DistanceRange{
        {Min: 0, Max: 500, A: 0, B: 0},
        {Min: 500, Max: 1000, A: 100, B: 1},
        {Min: 1000, Max: 0, A: 0, B: 0},
    }
    // in first range
    if r, ok, blocked := selectRange(rs, 200); !ok || blocked || r.Min != 0 || r.Max != 500 {
        t.Fatalf("range selection failed for 200")
    }
    // boundary at 500 -> second range
    if r, ok, blocked := selectRange(rs, 500); !ok || blocked || r.Min != 500 {
        t.Fatalf("range selection failed for 500")
    }
    // 999 still second
    if _, ok, blocked := selectRange(rs, 999); !ok || blocked {
        t.Fatalf("range selection failed for 999")
    }
    // 1000 blocked
    if _, ok, blocked := selectRange(rs, 1000); ok || !blocked {
        t.Fatalf("expected blocked at 1000")
    }
}

func TestPriceHandler_OK(t *testing.T) {
    f := &fakeClient{
        s: homeapi.StaticData{Lat: 0, Lon: 0, OrderMinimumNoSurcharge: 1000},
        d: homeapi.DynamicData{BasePrice: 199, DistanceRanges: []homeapi.DistanceRange{
            {Min: 0, Max: 500, A: 0, B: 0},
            {Min: 500, Max: 2000, A: 100, B: 1},
            {Min: 2000, Max: 0, A: 0, B: 0},
        }},
    }
    h := PriceHandler(f)
    // user at lon 0.005 deg (~556m at equator) to hit second range
    req := httptest.NewRequest(http.MethodGet, "/api/v1/delivery-order-price?venue_slug=v&cart_value=1000&user_lat=0&user_lon=0.005", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("status: %d body: %s", rr.Code, rr.Body.String())
    }
    var got struct{
        TotalPrice int `json:"total_price"`
        SmallOrderSurcharge int `json:"small_order_surcharge"`
        CartValue int `json:"cart_value"`
        Delivery struct{ Fee int `json:"fee"`; Distance int `json:"distance"` } `json:"delivery"`
    }
    if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
        t.Fatal(err)
    }
    if got.CartValue != 1000 || got.SmallOrderSurcharge != 0 {
        t.Fatalf("unexpected cart/surcharge: %+v", got)
    }
    if got.Delivery.Fee <= 199 { // second range adds something
        t.Fatalf("unexpected fee: %+v", got)
    }
    if got.TotalPrice != got.CartValue + got.SmallOrderSurcharge + got.Delivery.Fee {
        t.Fatalf("total mismatch")
    }
}

func TestPriceHandler_Blocked(t *testing.T) {
    f := &fakeClient{
        s: homeapi.StaticData{Lat: 0, Lon: 0, OrderMinimumNoSurcharge: 0},
        d: homeapi.DynamicData{BasePrice: 0, DistanceRanges: []homeapi.DistanceRange{
            {Min: 0, Max: 1000, A: 0, B: 0},
            {Min: 1000, Max: 0, A: 0, B: 0},
        }},
    }
    h := PriceHandler(f)
    // Place user far so distance >= 1000m. lon 0.01 deg ~1113m
    req := httptest.NewRequest(http.MethodGet, "/api/v1/delivery-order-price?venue_slug=v&cart_value=0&user_lat=0&user_lon=0.01", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d body: %s", rr.Code, rr.Body.String())
    }
}

func TestPriceHandler_Validation(t *testing.T) {
    f := &fakeClient{}
    h := PriceHandler(f)
    // Missing params
    req := httptest.NewRequest(http.MethodGet, "/api/v1/delivery-order-price", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", rr.Code)
    }
}

