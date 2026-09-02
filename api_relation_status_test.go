package api2go

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/manyminds/api2go/jsonapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// queueingItem is a minimal resource entity with an editable to-many relation,
// used to exercise what status a relationship edit answers with.
type queueingItem struct {
	ID         string `json:"-"`
	ThingIDs   []string
	addedIDs   []string
	deletedIDs []string
}

func (q queueingItem) GetID() string          { return q.ID }
func (q *queueingItem) SetID(id string) error { q.ID = id; return nil }
func (q queueingItem) GetName() string        { return "queueingItems" }

func (q queueingItem) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{{Type: "things", Name: "things"}}
}

func (q queueingItem) GetReferencedIDs() []jsonapi.ReferenceID {
	ids := []jsonapi.ReferenceID{}
	for _, id := range q.ThingIDs {
		ids = append(ids, jsonapi.ReferenceID{ID: id, Type: "things", Name: "things"})
	}
	return ids
}

func (q *queueingItem) AddToManyIDs(name string, IDs []string) error {
	q.addedIDs = append(q.addedIDs, IDs...)
	q.ThingIDs = append(q.ThingIDs, IDs...)
	return nil
}

func (q *queueingItem) DeleteToManyIDs(name string, IDs []string) error {
	q.deletedIDs = append(q.deletedIDs, IDs...)
	return nil
}

func (q *queueingItem) SetToManyReferenceIDs(name string, IDs []string) error {
	q.ThingIDs = IDs
	return nil
}

// queueingItemSource answers every Update with updateCode, standing in for a
// resource that queues background work off the back of a relationship edit and
// reports it through the Responder. refuse, when set, is returned from
// RefuseRelationEdit to stand in for a relation the resource cannot edit.
type queueingItemSource struct {
	updateCode int
	refuse     error
	updated    bool
}

func (s *queueingItemSource) RefuseRelationEdit(relation string, method string) error {
	return s.refuse
}

func (s *queueingItemSource) FindOne(id string, req Request) (Responder, error) {
	return &Response{Res: queueingItem{ID: id}}, nil
}

func (s *queueingItemSource) Update(obj interface{}, req Request) (Responder, error) {
	s.updated = true
	return &Response{Res: obj, Code: s.updateCode}, nil
}

var _ = Describe("Relationship edit status", func() {
	var (
		api    *API
		source *queueingItemSource
	)

	// edit issues one relationship edit and reports the status it answered with.
	edit := func(method string) int {
		body := strings.NewReader(`{"data": [{"type": "things", "id": "1"}]}`)
		rec := httptest.NewRecorder()
		req, err := http.NewRequest(method, "/v1/queueingItems/my-item/relationships/things", body)
		Expect(err).To(BeNil())
		api.Handler().ServeHTTP(rec, req)
		return rec.Code
	}

	BeforeEach(func() {
		source = &queueingItemSource{updateCode: http.StatusOK}
		api = NewAPIWithRouting(testPrefix, NewStaticResolver(""), newTestRouter())
		api.AddResource(queueingItem{}, source)
	})

	// A relationship edit has no body to return, so 204 is what it has always
	// answered and what it must keep answering for a resource that applies the
	// edit inline.
	It("answers 204 when the resource reports an ordinary success", func() {
		Expect(edit("POST")).To(Equal(http.StatusNoContent))
		Expect(edit("DELETE")).To(Equal(http.StatusNoContent))
		Expect(edit("PATCH")).To(Equal(http.StatusNoContent))
	})

	// The handlers used to discard the Responder and write 204 unconditionally,
	// so a resource that queued a background job had no way to say so and the
	// caller was handed a "nothing to report" for work still in flight.
	It("answers 202 when the resource reports it queued something", func() {
		source.updateCode = http.StatusAccepted
		Expect(edit("POST")).To(Equal(http.StatusAccepted))
		Expect(edit("DELETE")).To(Equal(http.StatusAccepted))
		Expect(edit("PATCH")).To(Equal(http.StatusAccepted))
	})

	// A resource routes every relationship edit through the same Update, so one
	// that applies the change for only some relations would otherwise answer 204
	// for the rest while discarding them.
	It("refuses an edit the resource declines, without calling Update", func() {
		source.refuse = NewHTTPError(nil, "cannot edit things this way", http.StatusForbidden)

		for _, method := range []string{"POST", "DELETE", "PATCH"} {
			source.updated = false
			Expect(edit(method)).To(Equal(http.StatusForbidden))
			Expect(source.updated).To(BeFalse(), "Update ran for a refused "+method)
		}
	})

	It("carries the resource's refusal message to the client", func() {
		source.refuse = NewHTTPError(nil, "cannot edit things this way", http.StatusForbidden)

		rec := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/v1/queueingItems/my-item/relationships/things",
			strings.NewReader(`{"data": [{"type": "things", "id": "1"}]}`))
		Expect(err).To(BeNil())
		api.Handler().ServeHTTP(rec, req)

		Expect(rec.Body.String()).To(MatchJSON(`{"errors":[{"status":"403","title":"Forbidden","detail":"cannot edit things this way"}]}`))
	})
})
