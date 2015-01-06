// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/cloudfoundry/bosh-micro-cli/cpi (interfaces: Deployment,DeploymentFactory)

package mocks

import (
	gomock "code.google.com/p/gomock/gomock"
	cloud "github.com/cloudfoundry/bosh-micro-cli/cloud"
	cpi "github.com/cloudfoundry/bosh-micro-cli/cpi"
	manifest "github.com/cloudfoundry/bosh-micro-cli/deployment/manifest"
)

// Mock of Deployment interface
type MockDeployment struct {
	ctrl     *gomock.Controller
	recorder *_MockDeploymentRecorder
}

// Recorder for MockDeployment (not exported)
type _MockDeploymentRecorder struct {
	mock *MockDeployment
}

func NewMockDeployment(ctrl *gomock.Controller) *MockDeployment {
	mock := &MockDeployment{ctrl: ctrl}
	mock.recorder = &_MockDeploymentRecorder{mock}
	return mock
}

func (_m *MockDeployment) EXPECT() *_MockDeploymentRecorder {
	return _m.recorder
}

func (_m *MockDeployment) Install() (cloud.Cloud, error) {
	ret := _m.ctrl.Call(_m, "Install")
	ret0, _ := ret[0].(cloud.Cloud)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockDeploymentRecorder) Install() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Install")
}

func (_m *MockDeployment) Manifest() manifest.CPIDeploymentManifest {
	ret := _m.ctrl.Call(_m, "Manifest")
	ret0, _ := ret[0].(manifest.CPIDeploymentManifest)
	return ret0
}

func (_mr *_MockDeploymentRecorder) Manifest() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Manifest")
}

func (_m *MockDeployment) StartJobs() error {
	ret := _m.ctrl.Call(_m, "StartJobs")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockDeploymentRecorder) StartJobs() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StartJobs")
}

func (_m *MockDeployment) StopJobs() error {
	ret := _m.ctrl.Call(_m, "StopJobs")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockDeploymentRecorder) StopJobs() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StopJobs")
}

// Mock of DeploymentFactory interface
type MockDeploymentFactory struct {
	ctrl     *gomock.Controller
	recorder *_MockDeploymentFactoryRecorder
}

// Recorder for MockDeploymentFactory (not exported)
type _MockDeploymentFactoryRecorder struct {
	mock *MockDeploymentFactory
}

func NewMockDeploymentFactory(ctrl *gomock.Controller) *MockDeploymentFactory {
	mock := &MockDeploymentFactory{ctrl: ctrl}
	mock.recorder = &_MockDeploymentFactoryRecorder{mock}
	return mock
}

func (_m *MockDeploymentFactory) EXPECT() *_MockDeploymentFactoryRecorder {
	return _m.recorder
}

func (_m *MockDeploymentFactory) NewDeployment(_param0 manifest.CPIDeploymentManifest, _param1 string, _param2 string) cpi.Deployment {
	ret := _m.ctrl.Call(_m, "NewDeployment", _param0, _param1, _param2)
	ret0, _ := ret[0].(cpi.Deployment)
	return ret0
}

func (_mr *_MockDeploymentFactoryRecorder) NewDeployment(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "NewDeployment", arg0, arg1, arg2)
}