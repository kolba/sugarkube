package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"testing"
)

func TestStackConfigVars(t *testing.T) {
	sc, err := vars.LoadStackConfig("large", "../vars/testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := Values{
		"provisioner": map[interface{}]interface{}{
			"memory":    4096,
			"cpus":      4,
			"disk_size": "120g",
		},
	}

	providerImpl, err := NewProvider(sc.Provider)
	assert.Nil(t, err)

	actual, err := StackConfigVars(providerImpl, sc)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}

func TestNewProviderError(t *testing.T) {
	actual, err := NewProvider("nonsense")
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewLocalProvider(t *testing.T) {
	actual, err := NewProvider(LOCAL)
	assert.Nil(t, err)
	assert.Equal(t, LocalProvider{}, actual)
}

func TestNewAWSProvider(t *testing.T) {
	actual, err := NewProvider(AWS)
	assert.Nil(t, err)
	assert.Equal(t, AwsProvider{}, actual)
}