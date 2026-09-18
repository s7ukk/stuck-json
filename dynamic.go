package stuckjson

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Node wraps parsed arbitrary JSON data and provides chainable, panic-free path traversal.

type Node struct {
	raw any
}

// Parse unmarshals JSON bytes into a dynamic queryable Node.

func Parse(data []byte) (*Node, error) {
	var val any

	if err := json.Unmarshal(data, &val); err != nil {
		return &Node{raw: nil}, err
	}

	return &Node{raw: val}, nil
}

// ParseString parses a JSON string into a dynamic queryable Node.

func ParseString(s string) (*Node, error) {
	return Parse([]byte(s))
}

// Wrap creates a Node directly from any Go value.

func Wrap(v any) *Node {
	return &Node{raw: v}
}

// Raw returns the underlying raw interface{} value.

func (n *Node) Raw() any {
	if n == nil {
		return nil
	}
	return n.raw
}

// Exists returns true if the node contains a non-nil value.

func (n *Node) Exists() bool {
	return n != nil && n.raw != nil
}

// Get navigates into nested keys using dot notation (e.g. "user.profile.name" or "items.0.id").

func (n *Node) Get(path string) *Node {
	if n == nil || n.raw == nil {
		return &Node{raw: nil}
	}

	keys := strings.Split(path, ".")
	current := n.raw

	for _, key := range keys {
		if current == nil {
			return &Node{raw: nil}
		}

		switch v := current.(type) {
		case map[string]any:
			current = v[key]
		case []any:
			idx, err := strconv.Atoi(key)

			if err != nil || idx < 0 || idx >= len(v) {
				return &Node{raw: nil}
			}

			current = v[idx]
		default:
			return &Node{raw: nil}
		}
	}

	return &Node{raw: current}
}

// String returns the string representation, or empty string if not a string.

func (n *Node) String() string {
	if n == nil || n.raw == nil {
		return ""
	}
	switch v := n.raw.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// Int returns int value, parsing from float/string if needed.

func (n *Node) Int() int {
	return int(n.Int64())
}

// Int64 safely returns int64 value.

func (n *Node) Int64() int64 {
	if n == nil || n.raw == nil {
		return 0
	}

	switch v := n.raw.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	default:
		return 0
	}
}

// Float64 safely returns float64 value.
func (n *Node) Float64() float64 {
	if n == nil || n.raw == nil {
		return 0
	}
	switch v := n.raw.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// Bool safely returns bool value.

func (n *Node) Bool() bool {
	if n == nil || n.raw == nil {
		return false
	}

	switch v := n.raw.(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	case float64:
		return v != 0
	default:
		return false
	}
}

// Slice returns array of child nodes if this node is an array.

func (n *Node) Slice() []*Node {
	if n == nil || n.raw == nil {
		return nil
	}

	arr, ok := n.raw.([]any)

	if !ok {
		return nil
	}

	res := make([]*Node, len(arr))

	for i, item := range arr {
		res[i] = &Node{raw: item}
	}

	return res
}

// StringSlice returns slice of strings.

func (n *Node) StringSlice() []string {
	nodes := n.Slice()

	if nodes == nil {
		return nil
	}

	out := make([]string, len(nodes))

	for i, node := range nodes {
		out[i] = node.String()
	}

	return out
}

// ToJSON marshals the node back to formatted or compact JSON string.

func (n *Node) ToJSON() (string, error) {
	if n == nil || n.raw == nil {
		return "null", nil
	}

	b, err := json.Marshal(n.raw)
	
	return string(b), err
}
