// Copyright 2020 Layer5.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"io"

	"github.com/gofrs/flock"
	"github.com/sirupsen/logrus"
)

// SafeClose is a helper function help to close the io
func SafeClose(co io.Closer, err *error) {
	if cerr := co.Close(); cerr != nil && *err == nil {
		*err = cerr
		logrus.Error(cerr)
	}
}

// SafeUnLock help safely unlock the file and log the error to stoutput
func SafeUnLock(locker *flock.Flock, err *error) {
	if uerr := locker.Unlock(); uerr != nil && *err == nil {
		*err = uerr
		logrus.Error(uerr)
	}
}
