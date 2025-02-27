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

package bolt

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/packstream"
)

func TestDehydrateHydrate(ot *testing.T) {
	hydrator := hydrator{}

	// A bit of white box testing, uses "internal" APIs to shortcut
	// hydration/dehydration circuit.
	dehydrateAndHydrate := func(t *testing.T, xi any) any {
		out := &outgoing{
			chunker: newChunker(),
			packer:  packstream.Packer{},
			onPackErr: func(err error) {
				ot.Fatalf("Should be no dehydration errors in this test: %s", err)
			},
			onIoErr: func(_ context.Context, err error) {
				ot.Fatalf("Should be no io errors in this test: %s", err)
			},
		}
		serv, cli := net.Pipe()
		defer func() {
			if err := cli.Close(); err != nil {
				ot.Errorf("failed to close client connection %v", err)
			}
			if err := serv.Close(); err != nil {
				ot.Errorf("failed to close server connection %v", err)
			}
		}()
		// format data in a record to avoid confusing the hydrator
		out.appendX(msgRecord, []any{xi})
		go func() {
			out.send(context.Background(), cli)
		}()
		_, byts, err := dechunkMessage(context.Background(), serv, []byte{}, -1)
		if err != nil {
			t.Fatal(err)
		}

		recx, err := hydrator.hydrate(byts)
		if err != nil {
			ot.Fatalf("Should be no hydration errors in this test: %s", err)
		}
		rec := recx.(*db.Record)
		return rec.Values[0]
	}

	ot.Run("time.Time", func(t *testing.T) {
		ni := time.Now()
		l, _ := time.LoadLocation("America/New_York")
		ni = ni.In(l)
		no := dehydrateAndHydrate(t, ni).(time.Time)
		assertDateTimeSame(t, ni, no)
		assertTimeLocationSame(t, ni, no)
	})

	ot.Run("time.Time offset", func(t *testing.T) {
		ni := time.Now()
		l := time.FixedZone("Offset", 60*60)
		ni = ni.In(l)
		no := dehydrateAndHydrate(t, ni).(time.Time)
		assertDateTimeSame(t, ni, no)
		assertTimeLocationSame(t, ni, no)
	})

	ot.Run("LocalDateTime", func(t *testing.T) {
		ni := time.Now().Round(0 * time.Nanosecond)
		l, _ := time.LoadLocation("America/New_York")
		ni = ni.In(l).Round(0 * time.Nanosecond)
		no := dehydrateAndHydrate(t, dbtype.LocalDateTime(ni)).(dbtype.LocalDateTime)
		assertTimeSame(t, ni, time.Time(no))
		assertDateSame(t, ni, time.Time(no))
		// Received time should be in Local time even if sent as something else
		if time.Time(no).Location().String() != "Local" {
			t.Errorf("Should be local")
		}
	})

	ot.Run("LocalDateTime way back", func(t *testing.T) {
		l, _ := time.LoadLocation("Asia/Anadyr")
		ni := time.Date(311, 7, 2, 23, 59, 3, 1, l)
		no := dehydrateAndHydrate(t, dbtype.LocalDateTime(ni)).(dbtype.LocalDateTime)
		assertTimeSame(t, ni, time.Time(no))
		assertDateSame(t, ni, time.Time(no))
		// Received time should be in Local time even if sent as something else
		if time.Time(no).Location().String() != "Local" {
			t.Errorf("Should be local")
		}
	})

	ot.Run("Date", func(t *testing.T) {
		ni := time.Now()
		l, _ := time.LoadLocation("America/New_York")
		ni = ni.In(l)
		no := dehydrateAndHydrate(t, dbtype.Date(ni)).(dbtype.Date)
		assertDateSame(t, ni, time.Time(no))
	})

	ot.Run("Time", func(t *testing.T) {
		ni := time.Now()
		l, _ := time.LoadLocation("America/New_York")
		ni = ni.In(l)
		no := dehydrateAndHydrate(t, dbtype.Time(ni)).(dbtype.Time)
		assertZoneOffsetSame(t, ni, time.Time(no))
		assertTimeSame(t, ni, time.Time(no))
	})

	ot.Run("LocalTime", func(t *testing.T) {
		ni := time.Now()
		l, _ := time.LoadLocation("America/New_York")
		ni = ni.In(l)
		no := dehydrateAndHydrate(t, dbtype.LocalTime(ni)).(dbtype.LocalTime)
		assertTimeSame(t, ni, time.Time(no))
	})

	ot.Run("Duration", func(t *testing.T) {
		di := dbtype.Duration{Months: 3, Days: 3, Seconds: 9000, Nanos: 13}
		do := dehydrateAndHydrate(t, di).(dbtype.Duration)
		assertDurationSame(t, di, do)
	})
}
