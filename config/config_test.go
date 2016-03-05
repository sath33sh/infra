package config

import (
	"testing"
)

func TestLoglevel(t *testing.T) {
	if Base.Get("loglevel") == nil {
		t.Errorf("Loglevel test failed")
	}
}
