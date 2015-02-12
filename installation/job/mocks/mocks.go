// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/cloudfoundry/bosh-micro-cli/installation/job (interfaces: Installer)

package mocks

import (
	gomock "code.google.com/p/gomock/gomock"
	job "github.com/cloudfoundry/bosh-micro-cli/installation/job"
	ui "github.com/cloudfoundry/bosh-micro-cli/ui"
)

// Mock of Installer interface
type MockInstaller struct {
	ctrl     *gomock.Controller
	recorder *_MockInstallerRecorder
}

// Recorder for MockInstaller (not exported)
type _MockInstallerRecorder struct {
	mock *MockInstaller
}

func NewMockInstaller(ctrl *gomock.Controller) *MockInstaller {
	mock := &MockInstaller{ctrl: ctrl}
	mock.recorder = &_MockInstallerRecorder{mock}
	return mock
}

func (_m *MockInstaller) EXPECT() *_MockInstallerRecorder {
	return _m.recorder
}

func (_m *MockInstaller) Install(_param0 job.RenderedJobRef, _param1 ui.Stage) (job.InstalledJob, error) {
	ret := _m.ctrl.Call(_m, "Install", _param0, _param1)
	ret0, _ := ret[0].(job.InstalledJob)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockInstallerRecorder) Install(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Install", arg0, arg1)
}
