/*
 * Copyright (c) 2002-2018 "Neo4j,"
 * Neo4j Sweden AB [http://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package connector_mocks

import neo4j "neo4j-go-connector"

type mockConnector struct {
	pool *mockPool
}

type mockPool struct {
	connection *MockConnection
}

func (connector *mockConnector) GetPool() (neo4j.Pool, error) {
	return connector.pool, nil
}

func (connector *mockConnector) Close() error {
	return connector.pool.Close()
}

func (pool *mockPool) Acquire() (neo4j.Connection, error) {
	return pool.connection, nil
}

func (pool *mockPool) Close() error {
	return pool.connection.Close()
}

func MockedConnector(connection *MockConnection) neo4j.Connector {
	return &mockConnector{
		pool: &mockPool{
			connection: connection,
		},
	}
}

