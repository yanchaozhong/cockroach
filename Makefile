# Copyright 2014 The Cockroach Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
# implied. See the License for the specific language governing
# permissions and limitations under the License. See the AUTHORS file
# for names of contributors.
#
# Author: Andrew Bonventre (andybons@gmail.com)
# Author: Shawn Morel (shawnmorel@gmail.com)
# Author: Spencer Kimball (spencer.kimball@gmail.com)

# Cockroach build rules.
GO ?= go

GOPATH      := $(CURDIR)/_vendor:$(GOPATH)
ROCKSDB     := $(CURDIR)/_vendor/rocksdb
ROACH_PROTO := $(CURDIR)/proto
ROACH_LIB   := $(CURDIR)/roachlib

CFLAGS   := "-I$(ROCKSDB)/include -I$(ROACH_PROTO)/lib -I$(ROACH_LIB) $(CFLAGS)"
CPPFLAGS := "-I$(ROCKSDB)/include -I$(ROACH_PROTO)/lib -I$(ROACH_LIB) $(CPPFLAGS)"
LDFLAGS  := "-L$(ROCKSDB) -L$(ROACH_PROTO)/lib -L$(ROACH_LIB) $(LDFLAGS)"

FLAGS := LDFLAGS=$(LDFLAGS) \
         CFLAGS=$(CFLAGS) \
         CPPFLAGS=$(CPPFLAGS)

CGO_FLAGS := CGO_LDFLAGS=$(LDFLAGS) \
             CGO_CFLAGS=$(CFLAGS) \
             CGO_CPPFLAGS=$(CPPFLAGS)

PKG       := "./..."
TESTS     := ".*"
TESTFLAGS := -logtostderr -timeout 10s

all: build test

build: rocksdb roach_proto roach_lib
	$(CGO_FLAGS) $(GO) build -o cockroach

rocksdb:
	cd $(ROCKSDB); make static_lib

roach_proto:
	cd $(ROACH_PROTO); $(FLAGS) make static_lib

roach_lib:
	cd $(ROACH_LIB); $(FLAGS) make static_lib

goget:
	$(CGO_FLAGS) $(GO) get ./...

test: rocksdb
	$(CGO_FLAGS) $(GO) test -run $(TESTS) $(PKG) $(TESTFLAGS)

testrace: rocksdb
	$(CGO_FLAGS) $(GO) test -race -run $(TESTS) $(PKG) $(TESTFLAGS)

coverage: rocksdb
	$(CGO_FLAGS) $(GO) test -cover -run $(TESTS) $(PKG) $(TESTFLAGS)

clean:
	$(GO) clean
	cd $(ROCKSDB); make clean
	cd $(ROACH_PROTO); make clean
