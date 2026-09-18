package stuckjson

import (
	"encoding/json"
	"testing"
)

type BenchData struct {
	ID     string   `json:"id"`
	Title  string   `json:"title" validate:"required,min=3"`
	Count  int      `json:"count" validate:"min=1"`
	Tags   []string `json:"tags"`
	Active bool     `json:"active"`
}

var sampleJSON = []byte(`{"id":"101","title":"Performance Test","count":42,"tags":["fast","go","json"],"active":true}`)

func BenchmarkStandardUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var d BenchData
		_ = json.Unmarshal(sampleJSON, &d)
	}
}

func BenchmarkStuckJSON_DecodeBytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeBytes[BenchData](sampleJSON)
	}
}

func BenchmarkStuckJSON_DecodeAndValidate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeAndValidate[BenchData](sampleJSON)
	}
}
