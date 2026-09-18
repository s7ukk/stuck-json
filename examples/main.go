package main

import (
	"fmt"
	"log"
	"net/http"

	stuckjson "github.com/senficode/stuck-json"
)

// CreateProductRequest defines the payload for creating a new product.

// Notice the concise tags: json, validate, and default!

type CreateProductRequest struct {
	SKU         string   `json:"sku" validate:"required,min=4,max=16"`
	Title       string   `json:"title" validate:"required,min=2,max=100"`
	Price       float64  `json:"price" validate:"min=0.01"`
	Category    string   `json:"category" validate:"oneof=tech|home|books|clothing" default:"tech"`
	Stock       int      `json:"stock" validate:"min=0" default:"10"`
	InStock     bool     `json:"in_stock" default:"true"`
	Description string   `json:"description" default:"No description provided"`
}

// Custom validation method

func (r *CreateProductRequest) Validate() error {
	if r.Price > 100000 && r.Stock > 100 {
		return stuckjson.FieldError{
			Field:   "stock",
			Rule:    "business_limit",
			Message: "luxury items with price > 100k cannot have stock > 100",
		}
	}

	return nil
}

func handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		stuckjson.WriteError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s not allowed", r.Method))
		return
	}

	// In just ONE line: binds JSON, sets default values, and validates all tags + custom logic !

	req, err := stuckjson.BindAndValidate[CreateProductRequest](r)

	if err != nil {
		// Automatically produces structured field errors in JSON response

		stuckjson.WriteError(w, http.StatusBadRequest, err)

		return
	}

	log.Printf("Successfully validated product: %+v", req)

	// One line to respond with JSON:

	_ = stuckjson.Write(w, http.StatusCreated, map[string]any{
		"status":  "created",
		"product": req,
	})
}

func main() {
	http.HandleFunc("/api/products", handleCreateProduct)

	port := ":8080"

	fmt.Printf("start http://localhost%s\n", port)
	fmt.Println("try POST /api/products with JSON payload:")
	fmt.Println(`curl -X POST http://localhost:8080/api/products -H "Content-Type: application/json" -d '{"sku":"PROD-123","title":"Go Keyboard Stuck","price":99.50}'`)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
