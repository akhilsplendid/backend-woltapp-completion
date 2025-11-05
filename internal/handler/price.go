package handler

import (
    "context"
    "encoding/json"
    "errors"
    "math"
    "net/http"
    "strconv"
    "time"

    "backend-woltapp-completion/internal/homeapi"
)

type priceResponse struct {
    TotalPrice         int           `json:"total_price"`
    SmallOrderSurcharge int          `json:"small_order_surcharge"`
    CartValue          int           `json:"cart_value"`
    Delivery           deliveryPart  `json:"delivery"`
}

type deliveryPart struct {
    Fee      int `json:"fee"`
    Distance int `json:"distance"`
}

type errorResponse struct {
    Error string `json:"error"`
}

// PriceHandler returns a handler that calculates delivery order price.
func PriceHandler(api homeapi.Client) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        q := r.URL.Query()
        venue := q.Get("venue_slug")
        cartStr := q.Get("cart_value")
        latStr := q.Get("user_lat")
        lonStr := q.Get("user_lon")

        if venue == "" || cartStr == "" || latStr == "" || lonStr == "" {
            writeError(w, http.StatusBadRequest, "missing required query parameters")
            return
        }
        cartValue, err := strconv.Atoi(cartStr)
        if err != nil || cartValue < 0 {
            writeError(w, http.StatusBadRequest, "cart_value must be a non-negative integer")
            return
        }
        userLat, err := strconv.ParseFloat(latStr, 64)
        if err != nil || userLat < -90 || userLat > 90 {
            writeError(w, http.StatusBadRequest, "user_lat must be a valid latitude")
            return
        }
        userLon, err := strconv.ParseFloat(lonStr, 64)
        if err != nil || userLon < -180 || userLon > 180 {
            writeError(w, http.StatusBadRequest, "user_lon must be a valid longitude")
            return
        }

        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        // Fetch static and dynamic in parallel
        type staticRes struct{ s homeapi.StaticData; err error }
        type dynamicRes struct{ d homeapi.DynamicData; err error }
        chStatic := make(chan staticRes, 1)
        chDynamic := make(chan dynamicRes, 1)

        go func() {
            s, e := api.GetStatic(ctx, venue)
            chStatic <- staticRes{s: s, err: e}
        }()
        go func() {
            d, e := api.GetDynamic(ctx, venue)
            chDynamic <- dynamicRes{d: d, err: e}
        }()

        var sdata homeapi.StaticData
        var ddata homeapi.DynamicData
        for i := 0; i < 2; i++ {
            select {
            case sr := <-chStatic:
                if sr.err != nil {
                    writeError(w, http.StatusBadRequest, "failed to fetch venue static info")
                    return
                }
                sdata = sr.s
            case dr := <-chDynamic:
                if dr.err != nil {
                    writeError(w, http.StatusBadRequest, "failed to fetch venue dynamic info")
                    return
                }
                ddata = dr.d
            case <-ctx.Done():
                writeError(w, http.StatusGatewayTimeout, "upstream timeout")
                return
            }
        }

        // Compute distance (meters)
        dist := haversineMeters(userLat, userLon, sdata.Lat, sdata.Lon)

        // Determine if delivery available and select range
        rng, ok, blocked := selectRange(ddata.DistanceRanges, dist)
        if blocked || !ok {
            writeError(w, http.StatusBadRequest, "delivery not available for this distance")
            return
        }

        // small order surcharge
        surcharge := sdata.OrderMinimumNoSurcharge - cartValue
        if surcharge < 0 {
            surcharge = 0
        }

        // fee = base + a + round(b * distance / 10)
        fee := ddata.BasePrice + rng.A + int(math.Round(float64(rng.B)*float64(dist)/10.0))
        total := cartValue + surcharge + fee

        writeJSON(w, http.StatusOK, priceResponse{
            TotalPrice:          total,
            SmallOrderSurcharge: surcharge,
            CartValue:           cartValue,
            Delivery:            deliveryPart{Fee: fee, Distance: dist},
        })
    })
}

// selectRange returns the selected distance range, ok if a priceable range was found,
// and blocked=true if a cutoff range (max==0) blocks delivery.
func selectRange(ranges []homeapi.DistanceRange, distance int) (homeapi.DistanceRange, bool, bool) {
    for _, r := range ranges {
        if r.Max == 0 && distance >= r.Min {
            return homeapi.DistanceRange{}, false, true
        }
        if distance >= r.Min && distance < r.Max {
            return r, true, false
        }
    }
    return homeapi.DistanceRange{}, false, false
}

// haversineMeters returns the great-circle distance in meters rounded to nearest int.
func haversineMeters(lat1, lon1, lat2, lon2 float64) int {
    const R = 6371000.0 // meters
    rad := func(d float64) float64 { return d * math.Pi / 180.0 }
    dlat := rad(lat2 - lat1)
    dlon := rad(lon2 - lon1)
    a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dlon/2)*math.Sin(dlon/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    return int(math.Round(R * c))
}

func writeError(w http.ResponseWriter, code int, msg string) {
    writeJSON(w, code, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    _ = json.NewEncoder(w).Encode(v)
}

// For tests
var (
    ErrBadRange = errors.New("bad range")
)

