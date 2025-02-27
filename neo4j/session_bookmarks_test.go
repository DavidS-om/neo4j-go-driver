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

package neo4j

import (
	"context"
	"reflect"
	"testing"
)

func TestSessionBookmarks(outer *testing.T) {
	ctx := context.Background()

	outer.Parallel()

	outer.Run("initial set bookmarks are cleaned up", func(t *testing.T) {
		sessionBookmarks := newSessionBookmarks(nil, []string{
			"", "bookmark", "", "deutschmark", "",
		})
		expectedBookmarks := []string{"bookmark", "deutschmark"}

		currentBookmarks := sessionBookmarks.currentBookmarks()

		if !reflect.DeepEqual(currentBookmarks, expectedBookmarks) {
			t.Errorf("expected bookmarks %v, got %v", expectedBookmarks, currentBookmarks)
		}
		lastBookmark := sessionBookmarks.lastBookmark()
		if lastBookmark != "deutschmark" {
			t.Errorf(`expected last bookmark "deutschmark", but got %q`, lastBookmark)
		}
	})

	outer.Run("replaces set bookmarks with new non-empty one", func(t *testing.T) {
		sessionBookmarks := newSessionBookmarks(nil, []string{
			"", "bookmark", "", "deutschmark", "",
		})

		err := sessionBookmarks.replaceBookmarks(ctx, nil, "booking mark")

		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		currentBookmarks := sessionBookmarks.currentBookmarks()
		if !reflect.DeepEqual(currentBookmarks, []string{"booking mark"}) {
			t.Errorf(`expected bookmarks ["booking mark"], got %v`, currentBookmarks)
		}
		lastBookmark := sessionBookmarks.lastBookmark()
		if lastBookmark != "booking mark" {
			t.Errorf(`expected last bookmark "booking mark", but got %q`, lastBookmark)
		}
	})

	outer.Run("does not replace set bookmarks when new bookmark is empty", func(t *testing.T) {
		sessionBookmarks := newSessionBookmarks(nil, []string{"book marking"})

		err := sessionBookmarks.replaceBookmarks(ctx, nil, "")

		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		currentBookmarks := sessionBookmarks.currentBookmarks()
		if !reflect.DeepEqual(currentBookmarks, []string{"book marking"}) {
			t.Errorf(`expected bookmarks ["book marking"], got %v`, currentBookmarks)
		}
		lastBookmark := sessionBookmarks.lastBookmark()
		if lastBookmark != "book marking" {
			t.Errorf(`expected last bookmark "book marking", but got %q`, lastBookmark)
		}
	})

	outer.Run("last bookmark returns empty string when no bookmarks were previously set", func(t *testing.T) {
		sessionBookmarks := newSessionBookmarks(nil, nil)

		lastBookmark := sessionBookmarks.lastBookmark()

		if lastBookmark != "" {
			t.Errorf(`expected empty last bookmark, but got %q`, lastBookmark)
		}
	})

	outer.Run("with bookmark manager", func(inner *testing.T) {
		inner.Run("notifies bookmark managers of bookmark updates", func(t *testing.T) {
			bookmarkManager := &fakeBookmarkManager{}
			sessionBookmarks := newSessionBookmarks(bookmarkManager, nil)

			err := sessionBookmarks.replaceBookmarks(ctx, []string{"b1", "b2"}, "b3")

			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if !bookmarkManager.called(1, "UpdateBookmarks", ctx, []string{"b1", "b2"}, []string{"b3"}) {
				t.Errorf("Expected UpdateBookmarks to be called once but was not")
			}
		})

		inner.Run("retrieves bookmarks", func(t *testing.T) {
			bookmarkManager := &fakeBookmarkManager{}
			sessionBookmarks := newSessionBookmarks(bookmarkManager, nil)

			_, _ = sessionBookmarks.getBookmarks(ctx)

			if !bookmarkManager.called(1, "GetBookmarks", ctx) {
				t.Errorf("Expected GetBookmarks with the provided arguments to be called once but was not")
			}
		})
	})
}

type invocation struct {
	function  string
	arguments []any
}

type fakeBookmarkManager struct {
	recordedCalls []invocation
}

func (f *fakeBookmarkManager) UpdateBookmarks(ctx context.Context, previousBookmarks, newBookmarks Bookmarks) error {
	f.recordedCalls = append(f.recordedCalls, invocation{
		function:  "UpdateBookmarks",
		arguments: []any{ctx, previousBookmarks, newBookmarks},
	})
	return nil
}

func (f *fakeBookmarkManager) GetBookmarks(ctx context.Context) (Bookmarks, error) {
	f.recordedCalls = append(f.recordedCalls, invocation{
		function:  "GetBookmarks",
		arguments: []any{ctx},
	})
	return nil, nil
}

func (f *fakeBookmarkManager) GetAllBookmarks(ctx context.Context) (Bookmarks, error) {
	f.recordedCalls = append(f.recordedCalls, invocation{
		function:  "GetAllBookmarks",
		arguments: []any{ctx},
	})
	return nil, nil
}

func (f *fakeBookmarkManager) Forget(context.Context) error {
	return nil
}

func (f *fakeBookmarkManager) called(times int, function string, args ...any) bool {
	count := 0
	for _, call := range f.recordedCalls {
		if call.function == function && reflect.DeepEqual(call.arguments, args) {
			count++
		}
	}
	return times == count
}
