/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package dbserver is used by integration tests to connect to databases
package dbserver

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
	"os"
	"strconv"
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var (
	mut    sync.Mutex
	server *DbServer
)

type DbServer struct {
	Username     string
	Password     string
	Scheme       string
	Hostname     string
	Port         int
	IsCluster    bool
	IsEnterprise bool
	Version      Version
}

func GetDbServer(ctx context.Context) DbServer {
	mut.Lock()
	defer mut.Unlock()

	if server == nil {
		vars := map[string]string{
			"TEST_NEO4J_HOST":       "",
			"TEST_NEO4J_USER":       "neo4j",
			"TEST_NEO4J_PASS":       "password",
			"TEST_NEO4J_SCHEME":     "neo4j",
			"TEST_NEO4J_PORT":       "7687",
			"TEST_NEO4J_EDITION":    "community",
			"TEST_NEO4J_IS_CLUSTER": "0",
			"TEST_NEO4J_VERSION":    "",
		}
		for k1, v1 := range vars {
			v2, e2 := os.LookupEnv(k1)
			if !e2 && v1 == "" {
				panic(fmt.Sprintf("Required environment variable %s is missing", k1))
			}
			if e2 {
				vars[k1] = v2
			}
		}
		key := "TEST_NEO4J_PORT"
		port, err := strconv.ParseUint(vars[key], 10, 16)
		if err != nil {
			panic(fmt.Sprintf("Unable to parse %s:%s to int", key, vars[key]))
		}
		key = "TEST_NEO4J_IS_CLUSTER"
		isCluster, err := strconv.ParseBool(vars[key])
		if err != nil {
			panic(fmt.Sprintf("Unable to parse %s:%s to bool", key, vars[key]))
		}
		server = &DbServer{
			Username:     vars["TEST_NEO4J_USER"],
			Password:     vars["TEST_NEO4J_PASS"],
			Scheme:       vars["TEST_NEO4J_SCHEME"],
			Hostname:     vars["TEST_NEO4J_HOST"],
			Port:         int(port),
			IsCluster:    isCluster,
			IsEnterprise: vars["TEST_NEO4J_EDITION"] == "enterprise",
			Version:      VersionOf(vars["TEST_NEO4J_VERSION"]),
		}
		server.deleteData(ctx)
	}
	return *server
}

func (s DbServer) deleteData(ctx context.Context) {
	driver := s.Driver()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	for {
		result, err := session.Run(ctx, "MATCH (n) WITH n LIMIT 10000 DETACH DELETE n RETURN count(n)", nil)
		if err != nil {
			panic(err)
		}

		if result.Next(ctx) {
			deleted := result.Record().Values[0].(int64)
			if deleted == 0 {
				break
			}
		}

		if err := result.Err(); err != nil {
			panic(err)
		}
	}
}

// URI returns the default URI to connect to the datbase.
// This should be used when tests don't care about the specifics of different URI schemes.
func (s DbServer) URI() string {
	return fmt.Sprintf("%s://%s:%d", s.Scheme, s.Hostname, s.Port)
}

func (s DbServer) BoltURI() string {
	return fmt.Sprintf("bolt://%s:%d", s.Hostname, s.Port)
}

// Returns the default auth token to connect to the database.
// This should be used when tests don't care about exactly what type of authorization scheme
// that is being used.
func (s DbServer) AuthToken() neo4j.AuthToken {
	return neo4j.BasicAuth(s.Username, s.Password, "")
}

func (s DbServer) Driver(configurers ...func(*config.Config)) neo4j.DriverWithContext {
	driver, err := neo4j.NewDriverWithContext(s.URI(), s.AuthToken(), configurers...)
	if err != nil {
		panic(err)
	}
	return driver
}

func (s DbServer) ConfigFunc() func(*config.Config) {
	return func(*config.Config) {}
}

func (s DbServer) CreateDatabaseQuery(db string) string {
	v := s.Version
	if s.isV42OrLater(v) {
		return fmt.Sprintf("CREATE DATABASE %s WAIT", db)
	}
	return fmt.Sprintf("CREATE DATABASE %s", db)
}

func (s DbServer) DropDatabaseQuery(db string) string {
	v := s.Version
	if s.isV42OrLater(v) {
		return fmt.Sprintf("DROP DATABASE %s IF EXISTS WAIT", db)
	}
	return fmt.Sprintf("DROP DATABASE %s IF EXISTS", db)
}

func (s DbServer) isV42OrLater(v Version) bool {
	return (v.major == 4 && v.minor >= 2) || v.major > 4
}

func (s DbServer) GetTransactionWorkloadsQuery() string {
	version := s.Version
	if version.LessThan(VersionOf("4.4.0")) {
		return "CALL dbms.listTransactions() YIELD status, currentQuery WHERE status = 'Running' RETURN currentQuery AS query"
	}
	return "SHOW TRANSACTIONS YIELD status, currentQuery WHERE status = 'Running' RETURN currentQuery AS query"
}
