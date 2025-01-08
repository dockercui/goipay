// Code generated by mockery v2.50.4. DO NOT EDIT.

package listener

import mock "github.com/stretchr/testify/mock"

// MockSharedTx is an autogenerated mock type for the SharedTx type
type MockSharedTx struct {
	mock.Mock
}

// GetConfirmations provides a mock function with no fields
func (_m *MockSharedTx) GetConfirmations() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetConfirmations")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetTxId provides a mock function with no fields
func (_m *MockSharedTx) GetTxId() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetTxId")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// IsDoubleSpendSeen provides a mock function with no fields
func (_m *MockSharedTx) IsDoubleSpendSeen() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsDoubleSpendSeen")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// NewMockSharedTx creates a new instance of MockSharedTx. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSharedTx(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSharedTx {
	mock := &MockSharedTx{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
