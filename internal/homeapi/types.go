package homeapi

// StaticData contains only the fields DOPC needs from the static endpoint.
type StaticData struct {
    Lat  float64
    Lon  float64
    OrderMinimumNoSurcharge int
}

// DistanceRange describes a pricing range based on distance.
type DistanceRange struct {
    Min int
    Max int // exclusive; 0 means delivery not available for >= Min
    A   int
    B   int
}

// DynamicData contains only the fields DOPC needs from the dynamic endpoint.
type DynamicData struct {
    BasePrice      int
    DistanceRanges []DistanceRange
}

