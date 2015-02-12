// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/cloudfoundry/bosh-micro-cli/deployment/disk (interfaces: Disk,Manager)

package mocks

import (
	gomock "code.google.com/p/gomock/gomock"
	property "github.com/cloudfoundry/bosh-micro-cli/common/property"
	disk "github.com/cloudfoundry/bosh-micro-cli/deployment/disk"
	manifest "github.com/cloudfoundry/bosh-micro-cli/deployment/manifest"
	ui "github.com/cloudfoundry/bosh-micro-cli/ui"
)

// Mock of Disk interface
type MockDisk struct {
	ctrl     *gomock.Controller
	recorder *_MockDiskRecorder
}

// Recorder for MockDisk (not exported)
type _MockDiskRecorder struct {
	mock *MockDisk
}

func NewMockDisk(ctrl *gomock.Controller) *MockDisk {
	mock := &MockDisk{ctrl: ctrl}
	mock.recorder = &_MockDiskRecorder{mock}
	return mock
}

func (_m *MockDisk) EXPECT() *_MockDiskRecorder {
	return _m.recorder
}

func (_m *MockDisk) CID() string {
	ret := _m.ctrl.Call(_m, "CID")
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockDiskRecorder) CID() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CID")
}

func (_m *MockDisk) Delete() error {
	ret := _m.ctrl.Call(_m, "Delete")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockDiskRecorder) Delete() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Delete")
}

func (_m *MockDisk) NeedsMigration(_param0 int, _param1 property.Map) bool {
	ret := _m.ctrl.Call(_m, "NeedsMigration", _param0, _param1)
	ret0, _ := ret[0].(bool)
	return ret0
}

func (_mr *_MockDiskRecorder) NeedsMigration(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "NeedsMigration", arg0, arg1)
}

// Mock of Manager interface
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *_MockManagerRecorder
}

// Recorder for MockManager (not exported)
type _MockManagerRecorder struct {
	mock *MockManager
}

func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &_MockManagerRecorder{mock}
	return mock
}

func (_m *MockManager) EXPECT() *_MockManagerRecorder {
	return _m.recorder
}

func (_m *MockManager) Create(_param0 manifest.DiskPool, _param1 string) (disk.Disk, error) {
	ret := _m.ctrl.Call(_m, "Create", _param0, _param1)
	ret0, _ := ret[0].(disk.Disk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockManagerRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Create", arg0, arg1)
}

func (_m *MockManager) DeleteUnused(_param0 ui.Stage) error {
	ret := _m.ctrl.Call(_m, "DeleteUnused", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockManagerRecorder) DeleteUnused(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "DeleteUnused", arg0)
}

func (_m *MockManager) FindCurrent() ([]disk.Disk, error) {
	ret := _m.ctrl.Call(_m, "FindCurrent")
	ret0, _ := ret[0].([]disk.Disk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockManagerRecorder) FindCurrent() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindCurrent")
}

func (_m *MockManager) FindUnused() ([]disk.Disk, error) {
	ret := _m.ctrl.Call(_m, "FindUnused")
	ret0, _ := ret[0].([]disk.Disk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockManagerRecorder) FindUnused() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUnused")
}
