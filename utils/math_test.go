package utils

import "testing"

func TestAdd(t *testing.T) {
	result := Add(5, 3)
	expected := 8
	if result != expected {
		t.Errorf("Add(2, 3) = %d; want %d", result, expected)
	}
}

func TestMultiply(t *testing.T) {
	result := Multiply(5, 3)
	expected := 15
	if result != expected {
		t.Errorf("Multiply(2, 3) = %d; want %d", result, expected)
	}
}
