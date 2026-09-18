package stuckjson

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// Validatable is an optional interface structs can implement for custom business validation.

type Validatable interface {
	Validate() error
}

var (
	uuidRegex   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	cachedTypes sync.Map // map[reflect.Type][]cachedFieldInfo
)

type cachedFieldInfo struct {
	index     int
	jsonName  string
	rules     []string
	isNested  bool
	isPointer bool
}

// ValidateStruct inspects struct tags (`validate:"..."`) and runs all validation rules.

// If the struct implements Validatable, its custom Validate() method is executed as well.

func ValidateStruct(v any) error {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("stuckjson.ValidateStruct: expected struct, got %s", val.Kind())
	}

	var errors ValidationErrors

	collectStructErrors(val, &errors)

	// Check if the struct implements Validatable

	if validatable, ok := v.(Validatable); ok {
		if err := validatable.Validate(); err != nil {
			if ve, ok := err.(ValidationErrors); ok {
				errors = append(errors, ve...)
			} else if fe, ok := err.(FieldError); ok {
				errors = append(errors, fe)
			} else {
				errors = append(errors, FieldError{
					Field:   "_",
					Rule:    "custom",
					Message: err.Error(),
				})
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func collectStructErrors(val reflect.Value, errs *ValidationErrors) {
	typ := val.Type()
	fields := getCachedFields(typ)

	for _, f := range fields {
		fieldVal := val.Field(f.index)

		if f.isNested {
			if f.isPointer {
				if !fieldVal.IsNil() {
					collectStructErrors(fieldVal.Elem(), errs)
				}
			} else {
				collectStructErrors(fieldVal, errs)
			}
			continue
		}

		for _, rule := range f.rules {
			if err := checkRule(f.jsonName, fieldVal, rule); err != nil {
				*errs = append(*errs, *err)
			}
		}
	}
}

func getCachedFields(typ reflect.Type) []cachedFieldInfo {
	if cached, ok := cachedTypes.Load(typ); ok {
		return cached.([]cachedFieldInfo)
	}

	var list []cachedFieldInfo

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)

		if !sf.IsExported() {
			continue
		}

		jsonTag := sf.Tag.Get("json")
		jsonName := sf.Name

		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")

			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				jsonName = parts[0]
			}
		}

		vTag := sf.Tag.Get("validate")
		isStruct := sf.Type.Kind() == reflect.Struct
		isPtrStruct := sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct

		var rules []string

		if vTag != "" {
			rules = strings.Split(vTag, ",")
		}

		list = append(list, cachedFieldInfo{
			index:     i,
			jsonName:  jsonName,
			rules:     rules,
			isNested:  isStruct || isPtrStruct,
			isPointer: isPtrStruct,
		})
	}

	cachedTypes.Store(typ, list)

	return list
}

func checkRule(fieldName string, val reflect.Value, rule string) *FieldError {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil
	}

	var ruleName string
	var ruleParam string
	if idx := strings.IndexByte(rule, '='); idx != -1 {
		ruleName = rule[:idx]
		ruleParam = rule[idx+1:]
	} else {
		ruleName = rule
	}

	isFieldZero := isZero(val)

	// 'required' is checked regardless of whether the field is zero

	if ruleName == "required" {
		if isFieldZero {
			return &FieldError{
				Field:   fieldName,
				Rule:    "required",
				Message: fmt.Sprintf("поле '%s' обязательно для заполнения", fieldName),
			}
		}

		return nil
	}

	// If field is zero and not required, other rules are skipped

	if isFieldZero {
		return nil
	}

	switch ruleName {
	case "email":
		str := val.String()

		_, err := mail.ParseAddress(str)

		if err != nil || !strings.Contains(str, ".") {
			return &FieldError{
				Field:   fieldName,
				Rule:    "email",
				Message: fmt.Sprintf("поле '%s' должно содержать корректный email адрес", fieldName),
				Value:   str,
			}
		}

	case "url":
		str := val.String()

		u, err := url.ParseRequestURI(str)

		if err != nil || u.Scheme == "" || u.Host == "" {
			return &FieldError{
				Field:   fieldName,
				Rule:    "url",
				Message: fmt.Sprintf("поле '%s' должно содержать валидный URL (начиная с http:// или https://)", fieldName),
				Value:   str,
			}
		}

	case "uuid":
		str := val.String()

		if !uuidRegex.MatchString(str) {
			return &FieldError{
				Field:   fieldName,
				Rule:    "uuid",
				Message: fmt.Sprintf("поле '%s' должно быть в формате UUIDv4", fieldName),
				Value:   str,
			}
		}

	case "min":
		n, _ := strconv.ParseFloat(ruleParam, 64)

		switch val.Kind() {
		case reflect.String:
			if float64(len([]rune(val.String()))) < n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "min",
					Message: fmt.Sprintf("минимальная длина поля '%s' — %s симв.", fieldName, ruleParam),
					Value:   val.String(),
				}
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(val.Int()) < n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "min",
					Message: fmt.Sprintf("значение поля '%s' должно быть не меньше %s", fieldName, ruleParam),
					Value:   val.Int(),
				}
			}

		case reflect.Float32, reflect.Float64:
			if val.Float() < n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "min",
					Message: fmt.Sprintf("значение поля '%s' должно быть не меньше %s", fieldName, ruleParam),
					Value:   val.Float(),
				}
			}

		case reflect.Slice:
			if float64(val.Len()) < n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "min",
					Message: fmt.Sprintf("поле '%s' должно содержать минимум %s элемент(ов)", fieldName, ruleParam),
				}
			}
		}

	case "max":
		n, _ := strconv.ParseFloat(ruleParam, 64)

		switch val.Kind() {
		case reflect.String:
			if float64(len([]rune(val.String()))) > n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "max",
					Message: fmt.Sprintf("максимальная длина поля '%s' — %s симв.", fieldName, ruleParam),
					Value:   val.String(),
				}
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(val.Int()) > n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "max",
					Message: fmt.Sprintf("значение поля '%s' должно быть не больше %s", fieldName, ruleParam),
					Value:   val.Int(),
				}
			}

		case reflect.Float32, reflect.Float64:
			if val.Float() > n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "max",
					Message: fmt.Sprintf("значение поля '%s' должно быть не больше %s", fieldName, ruleParam),
					Value:   val.Float(),
				}
			}

		case reflect.Slice:
			if float64(val.Len()) > n {
				return &FieldError{
					Field:   fieldName,
					Rule:    "max",
					Message: fmt.Sprintf("поле '%s' должно содержать не более %s элемент(ов)", fieldName, ruleParam),
				}
			}
		}

	case "oneof", "in":
		allowed := strings.Split(ruleParam, "|")

		if len(allowed) == 1 {
			allowed = strings.Split(ruleParam, ";")
		}

		strVal := fmt.Sprint(val.Interface())
		matched := false

		for _, a := range allowed {
			if strings.TrimSpace(a) == strVal {
				matched = true
				break
			}
		}

		if !matched {
			return &FieldError{
				Field:   fieldName,
				Rule:    "oneof",
				Message: fmt.Sprintf("поле '%s' должно принимать одно из значений: [%s]", fieldName, strings.Join(allowed, ", ")),
				Value:   strVal,
			}
		}

	case "alphanumeric":
		str := val.String()

		for _, r := range str {
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
				return &FieldError{
					Field:   fieldName,
					Rule:    "alphanumeric",
					Message: fmt.Sprintf("поле '%s' может содержать только буквы и цифры", fieldName),
					Value:   str,
				}
			}
		}

	case "json":
		str := val.String()

		if !json.Valid([]byte(str)) {
			return &FieldError{
				Field:   fieldName,
				Rule:    "json",
				Message: fmt.Sprintf("поле '%s' должно быть валидной строкой JSON", fieldName),
			}
		}
	}

	return nil
}

// FluentValidator provides a lightweight programmatic way to validate fields without tags.

type FluentValidator struct {
	errors ValidationErrors
}

// NewValidator creates a fresh programmatic validator.

func NewValidator() *FluentValidator {
	return &FluentValidator{}
}

// Required checks that val is not empty/zero.

func (v *FluentValidator) Required(field string, val any) *FluentValidator {
	if val == nil || isZero(reflect.ValueOf(val)) {
		v.errors = append(v.errors, FieldError{
			Field:   field,
			Rule:    "required",
			Message: fmt.Sprintf("поле '%s' обязательно для заполнения", field),
		})
	}

	return v
}

// Email validates email string.

func (v *FluentValidator) Email(field string, email string) *FluentValidator {
	if email == "" {
		return v
	}
	_, err := mail.ParseAddress(email)

	if err != nil || !strings.Contains(email, ".") {
		v.errors = append(v.errors, FieldError{
			Field:   field,
			Rule:    "email",
			Message: fmt.Sprintf("поле '%s' содержит невалидный email", field),
			Value:   email,
		})
	}

	return v
}

// MinLen checks minimum string length.

func (v *FluentValidator) MinLen(field string, s string, min int) *FluentValidator {
	if len([]rune(s)) < min {
		v.errors = append(v.errors, FieldError{
			Field:   field,
			Rule:    "min",
			Message: fmt.Sprintf("минимальная длина '%s' — %d симв.", field, min),
			Value:   s,
		})
	}

	return v
}

// RangeInt checks that an integer is between min and max inclusive.

func (v *FluentValidator) RangeInt(field string, val int, min, max int) *FluentValidator {
	if val < min || val > max {
		v.errors = append(v.errors, FieldError{
			Field:   field,
			Rule:    "range",
			Message: fmt.Sprintf("значение '%s' должно быть в диапазоне от %d до %d", field, min, max),
			Value:   val,
		})
	}

	return v
}

// OneOf checks string against a list of acceptable values.

func (v *FluentValidator) OneOf(field string, val string, allowed ...string) *FluentValidator {
	for _, a := range allowed {
		if a == val {
			return v
		}
	}

	v.errors = append(v.errors, FieldError{
		Field:   field,
		Rule:    "oneof",
		Message: fmt.Sprintf("значение '%s' должно быть одним из: %s", field, strings.Join(allowed, ", ")),
		Value:   val,
	})

	return v
}

// AddError manually appends a custom validation error.

func (v *FluentValidator) AddError(field, rule, message string) *FluentValidator {
	v.errors = append(v.errors, FieldError{
		Field:   field,
		Rule:    rule,
		Message: message,
	})

	return v
}

// Validate finishes validation and returns error if any rules failed, or nil.

func (v *FluentValidator) Validate() error {
	if len(v.errors) > 0 {
		return v.errors
	}
	
	return nil
}
