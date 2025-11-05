# DOPC (Delivery Order Price Calculator) — Go

[![CI](https://github.com/akhilsplendid/backend-woltapp-completion/actions/workflows/ci.yml/badge.svg)](https://github.com/akhilsplendid/backend-woltapp-completion/actions/workflows/ci.yml)

This is a Go implementation of the DOPC service described in the assignment README. It exposes one endpoint to compute delivery order pricing.

## Quick Start

- Requirements: Go 1.21+

Run the server:

```
cd backend-woltapp-completion
go run ./cmd/dopc
```

Environment variables:
- `PORT` (default: `8000`)
- `HOME_ASSIGNMENT_API_BASE` (default: `https://consumer-api.development.dev.woltapi.com`)

Example request:

```
curl "http://localhost:8000/api/v1/delivery-order-price?venue_slug=home-assignment-venue-helsinki&cart_value=1000&user_lat=60.17094&user_lon=24.93087"
```

## Endpoint

- `GET /api/v1/delivery-order-price`
  - Query params (all required):
    - `venue_slug` (string)
    - `cart_value` (integer >= 0)
    - `user_lat` (float, -90..90)
    - `user_lon` (float, -180..180)
  - Response 200 JSON:
    - `total_price`, `small_order_surcharge`, `cart_value`, `delivery.fee`, `delivery.distance`
  - Response 400 JSON (examples):
    - `{ "error": "missing required query parameters" }`
    - `{ "error": "delivery not available for this distance" }`

## Implementation Notes

- Fetches `static` and `dynamic` venue data in parallel from the Home Assignment API (base URL can be overridden via env var).
- Distance is calculated using the Haversine formula and rounded to nearest meter.
- Delivery availability and fee use `distance_ranges` as specified:
  - Select range where `min <= distance < max`.
  - If a range with `max == 0` has `distance >= min`, delivery is not available.
  - Fee: `base_price + a + round(b * distance / 10)`.
- `small_order_surcharge = max(0, order_minimum_no_surcharge - cart_value)`.

## Testing

```
cd backend-woltapp-completion
go test ./...
```

Tests cover:
- Haversine distance
- Range selection and cutoff behavior
- Handler happy path and error cases using a fake Home Assignment API client

## Makefile

Common tasks:

```
make run   # run locally
make test  # run tests
make build # build binary ./dopc
```

## Docker

Build the image:

```
cd backend-woltapp-completion
docker build -t dopc:local .
```

Run the container (port 8000):

```
docker run --rm -p 8000:8000 \
  -e HOME_ASSIGNMENT_API_BASE=https://consumer-api.development.dev.woltapi.com \
  --name dopc dopc:local
```

Sample request:

```
curl "http://localhost:8000/api/v1/delivery-order-price?venue_slug=home-assignment-venue-helsinki&cart_value=1000&user_lat=60.17094&user_lon=24.93087"
```
