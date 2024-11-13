package retry

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testableRetryFunc returns true if the returned error is a testableRetrierError and indicates that an action should be tried until the retrier hits its limit.
var testableRetryFunc = func(err error) bool {
	_, ok := err.(*testableRetrierError)
	return ok
}

// testableRetrierError marks errors that indicate that a previously executed action should be retried with again. It must wrap an existing error.
type testableRetrierError struct {
	Err error
}

// Error returns the error's string representation.
func (tre *testableRetrierError) Error() string {
	return tre.Err.Error()
}

func Test_OnErrorRetry(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		maxTries := 2
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return nil
		}

		// when
		err := OnError(maxTries, AlwaysRetryFunc, fn)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		maxTries := 2
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return assert.AnError
		}

		// when
		err := OnError(maxTries, AlwaysRetryFunc, fn)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_OnErrorWithLimit(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		limit := 2 * time.Millisecond
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return nil
		}

		t1 := time.Now()
		// when
		err := OnErrorWithLimit(limit, AlwaysRetryFunc, fn)
		t2 := time.Now()
		timeDiff := t2.Sub(t1)

		// then
		require.NoError(t, err)
		assert.Less(t, timeDiff, limit)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		limit := 3 * time.Second
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return assert.AnError
		}

		t1 := time.Now()
		// when
		err := OnErrorWithLimit(limit, AlwaysRetryFunc, fn)
		t2 := time.Now()
		timeDiff := t2.Sub(t1)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Greater(t, timeDiff, limit)
	})
}

func Test_OnConflict(t *testing.T) {
	t.Run("should retry once and succeed", func(t *testing.T) {
		// given
		retryCount := 0
		fn := func() error {
			retryCount++
			if retryCount == 1 {
				return &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonConflict}}
			}
			return nil
		}

		// when
		err := OnConflict(fn)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return assert.AnError
		}

		// when
		err := OnConflict(fn)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_testableRetrierError(t *testing.T) {
	sut := new(testableRetrierError)
	sut.Err = assert.AnError
	require.Error(t, sut)
	assert.ErrorContains(t, sut, assert.AnError.Error())
}

func Test_testableRetryFunc(t *testing.T) {
	assert.False(t, testableRetryFunc(nil))
	assert.False(t, testableRetryFunc(assert.AnError))
	retrierErr := new(testableRetrierError)
	retrierErr.Err = assert.AnError
	assert.True(t, testableRetryFunc(retrierErr))
}
