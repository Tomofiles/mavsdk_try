package main

type ThreeDModel struct {
	Id string `json:"id"`
	Name string `json:"name,omitempty"`
	Availability string `json:"availability,omitempty"`
	Position *Position `json:"position,omitempty"`
	Orientation *Orientation `json:"orientation,omitempty"`
	Path *Path `json:"path,omitempty"`
	Model *Model `json:"model,omitempty"`
}

type Path struct {
	Show bool `json:"show,omitempty"`
	Width int32 `json:"width,omitempty"`
	LeadTime int32 `json:"leadTime,omitempty"`
	Resolution float64 `json:"resolution,omitempty"`
	Material *Material `json:"material,omitempty"`
}

type Material struct {
	SolidColor *Color `json:"solidColor,omitempty"`
	Resolution int32 `json:"resolution,omitempty"`
}

type Color struct {
	Rgba []int32 `json:"rgba,omitempty"`
}

type Position struct {
	Epoch string `json:"epoch,omitempty"`
	CartographicDegrees []float64 `json:"cartographicDegrees"`
}

type Orientation struct {
	Epoch string `json:"epoch,omitempty"`
	UnitQuaternion []float64 `json:"unitQuaternion"`
}

type Model struct {
	Gltf string `json:"gltf,omitempty"`
	Scale float64 `json:"scale,omitempty"`
	MinimumPixelSize float64 `json:"minimumPixelSize,omitempty"`
	Show bool `json:"show,omitempty"`
}