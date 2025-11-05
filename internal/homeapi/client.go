package homeapi

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
)

// Client defines methods to fetch static and dynamic info.
type Client interface {
    GetStatic(ctx context.Context, venueSlug string) (StaticData, error)
    GetDynamic(ctx context.Context, venueSlug string) (DynamicData, error)
}

type client struct {
    baseURL string
    http    *http.Client
}

func New(baseURL string, httpClient *http.Client) Client {
    return &client{baseURL: baseURL, http: httpClient}
}

func (c *client) GetStatic(ctx context.Context, venueSlug string) (StaticData, error) {
    url := fmt.Sprintf("%s/home-assignment-api/v1/venues/%s/static", c.baseURL, venueSlug)
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    resp, err := c.http.Do(req)
    if err != nil {
        return StaticData{}, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
        return StaticData{}, fmt.Errorf("static endpoint %d: %s", resp.StatusCode, string(b))
    }
    var raw map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
        return StaticData{}, err
    }
    // Traverse known paths conservatively.
    vraw, _ := raw["venue_raw"].(map[string]any)
    if vraw == nil {
        return StaticData{}, errors.New("missing venue_raw")
    }
    // location can be GeoJSON-like with coordinates array [lon, lat],
    // or have direct lat/lon fields, or a coordinates object. Try all.
    loc, _ := vraw["location"].(map[string]any)
    var lat, lon float64
    if loc != nil {
        if coordsMap, ok := loc["coordinates"].(map[string]any); ok {
            lat, _ = toF64(coordsMap["lat"])
            lon, _ = toF64(coordsMap["lon"])
        } else if coordsArr, ok := loc["coordinates"].([]any); ok {
            if len(coordsArr) >= 2 {
                // GeoJSON order: [lon, lat]
                lon, _ = toF64(coordsArr[0])
                lat, _ = toF64(coordsArr[1])
            }
        } else {
            lat, _ = toF64(loc["lat"])
            lon, _ = toF64(loc["lon"])
        }
    }

    dspecs, _ := vraw["delivery_specs"].(map[string]any)
    minNoSurcharge := 0
    if dspecs != nil {
        minNoSurcharge = toInt(dspecs["order_minimum_no_surcharge"])
    }

    if lat == 0 && lon == 0 && minNoSurcharge == 0 {
        return StaticData{}, errors.New("incomplete static response")
    }
    return StaticData{Lat: lat, Lon: lon, OrderMinimumNoSurcharge: minNoSurcharge}, nil
}

func (c *client) GetDynamic(ctx context.Context, venueSlug string) (DynamicData, error) {
    url := fmt.Sprintf("%s/home-assignment-api/v1/venues/%s/dynamic", c.baseURL, venueSlug)
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    resp, err := c.http.Do(req)
    if err != nil {
        return DynamicData{}, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
        return DynamicData{}, fmt.Errorf("dynamic endpoint %d: %s", resp.StatusCode, string(b))
    }
    var raw map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
        return DynamicData{}, err
    }
    vraw, _ := raw["venue_raw"].(map[string]any)
    if vraw == nil {
        return DynamicData{}, errors.New("missing venue_raw")
    }
    dspecs, _ := vraw["delivery_specs"].(map[string]any)
    dpricing, _ := dspecs["delivery_pricing"].(map[string]any)
    base := toInt(dpricing["base_price"])
    var ranges []DistanceRange
    if drs, ok := dpricing["distance_ranges"].([]any); ok {
        for _, r := range drs {
            if rm, ok := r.(map[string]any); ok {
                ranges = append(ranges, DistanceRange{
                    Min: toInt(rm["min"]),
                    Max: toInt(rm["max"]),
                    A:   toInt(rm["a"]),
                    B:   toInt(rm["b"]),
                })
            }
        }
    }
    if base == 0 && len(ranges) == 0 {
        return DynamicData{}, errors.New("incomplete dynamic response")
    }
    return DynamicData{BasePrice: base, DistanceRanges: ranges}, nil
}

func toF64(v any) (float64, bool) {
    switch t := v.(type) {
    case float64:
        return t, true
    case float32:
        return float64(t), true
    case int:
        return float64(t), true
    case int64:
        return float64(t), true
    case json.Number:
        f, err := t.Float64()
        if err != nil {
            return 0, false
        }
        return f, true
    default:
        return 0, false
    }
}

func toInt(v any) int {
    switch t := v.(type) {
    case float64:
        return int(t)
    case float32:
        return int(t)
    case int:
        return t
    case int64:
        return int(t)
    case json.Number:
        i, err := t.Int64()
        if err == nil {
            return int(i)
        }
        // Fallback try float
        if f, err2 := t.Float64(); err2 == nil {
            return int(f)
        }
        return 0
    default:
        return 0
    }
}
