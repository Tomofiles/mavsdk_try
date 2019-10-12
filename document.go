package main

type Document struct {
	Id string `json:"id"`
	Version string `json:"version"`
	Clock *Clock `json:"clock"`
}

type Clock struct {
	Interval string `json:"interval"`
	CurrentTime string `json:"currentTime"`
	Multiplier float64 `json:"multiplier"`
	Range string `json:"range"`
	Step string `json:"step"`
}