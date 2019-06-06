# Copyright 2018 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang AS builder

# Get the dependencies from GitHub
RUN go get google.golang.org/grpc

WORKDIR /go/src/github.com/kwyn/knative-demo
ADD . /go/src/github.com/kwyn/knative-demo

RUN CGO_ENABLED=0 go build -tags=grpcping ./grpc-ping.go

FROM gcr.io/distroless/static

EXPOSE 8080
COPY --from=builder /go/src/git.rsglab.com/rsg/gke-services/knative-grpc-test/grpc-ping /server

ENTRYPOINT ["/server"]
