package mocks

import (
	"io"

	"github.com/compozed/deployadactyl/structs"
	"github.com/stretchr/testify/mock"
)

// Deployer is an autogenerated mock type for the Deployer type
type Deployer struct {
	mock.Mock
}

// Deploy provides a mock function with given fields: deploymentInfo, out
func (_m *Deployer) Deploy(deploymentInfo structs.DeploymentInfo, out io.Writer) error {
	ret := _m.Called(deploymentInfo, out)

	var r0 error
	if rf, ok := ret.Get(0).(func(structs.DeploymentInfo, io.Writer) error); ok {
		r0 = rf(deploymentInfo, out)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}