package api2go

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// createdID is the id every source below assigns, so the expected Location is
// the same string in each spec.
const createdID = "generated"

// readableItem is created by a source that also implements FindOne, so api2go
// registers an item route for it.
type readableItem struct {
	ID   string `json:"-"`
	Name string `json:"name"`
}

func (i readableItem) GetID() string          { return i.ID }
func (i *readableItem) SetID(id string) error { i.ID = id; return nil }
func (i readableItem) GetName() string        { return "readableItems" }

// actionItem stands in for an action resource: something a client asks for,
// not something it can read back.
type actionItem struct {
	ID   string `json:"-"`
	Name string `json:"name"`
}

func (i actionItem) GetID() string          { return i.ID }
func (i *actionItem) SetID(id string) error { i.ID = id; return nil }
func (i actionItem) GetName() string        { return "actionItems" }

// secretItem has an item route that always refuses the read.
type secretItem struct {
	ID   string `json:"-"`
	Name string `json:"name"`
}

func (i secretItem) GetID() string          { return i.ID }
func (i *secretItem) SetID(id string) error { i.ID = id; return nil }
func (i secretItem) GetName() string        { return "secretItems" }

// readableSource registers an item route, so its creates advertise a Location.
// A create naming itself "queued" answers 202 to cover the branch api2go
// short-circuits before marshaling.
type readableSource struct{}

func (s readableSource) FindOne(id string, req Request) (Responder, error) {
	return &Response{Res: readableItem{ID: id}}, nil
}

func (s readableSource) Create(obj interface{}, req Request) (Responder, error) {
	item := obj.(readableItem)
	code := http.StatusCreated
	if item.Name == "queued" {
		code = http.StatusAccepted
	}
	item.ID = createdID
	return &Response{Res: item, Code: code}, nil
}

// actionSource implements Create and nothing else, so api2go registers no item
// route for it.
type actionSource struct{}

func (s actionSource) Create(obj interface{}, req Request) (Responder, error) {
	item := obj.(actionItem)
	item.ID = createdID
	return &Response{Res: item, Code: http.StatusAccepted}, nil
}

// secretSource has an item route but declines the header.
type secretSource struct{}

func (s secretSource) FindOne(id string, req Request) (Responder, error) {
	return &Response{}, NewHTTPError(nil, "reading is not possible", http.StatusForbidden)
}

func (s secretSource) Create(obj interface{}, req Request) (Responder, error) {
	item := obj.(secretItem)
	item.ID = createdID
	return &Response{Res: item, Code: http.StatusCreated}, nil
}

func (s secretSource) SuppressLocationHeader() bool { return true }

var _ = Describe("Location header on create", func() {
	var api *API

	BeforeEach(func() {
		api = NewAPIWithRouting(testPrefix, NewStaticResolver(""), newTestRouter())
		api.AddResource(readableItem{}, readableSource{})
		api.AddResource(actionItem{}, actionSource{})
		api.AddResource(secretItem{}, secretSource{})
	})

	create := func(collection, name string) *httptest.ResponseRecorder {
		body := strings.NewReader(`{"data": {"type": "` + collection + `", "attributes": {"name": "` + name + `"}}}`)
		rec := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/v1/"+collection, body)
		Expect(err).To(BeNil())
		api.Handler().ServeHTTP(rec, req)
		return rec
	}

	It("sets Location on a 201 from a source with an item route", func() {
		rec := create("readableItems", "example")

		Expect(rec.Code).To(Equal(http.StatusCreated))
		Expect(rec.Header().Get("Location")).To(Equal("/v1/readableItems/" + createdID))
	})

	It("sets Location on a 202 from a source with an item route", func() {
		rec := create("readableItems", "queued")

		Expect(rec.Code).To(Equal(http.StatusAccepted))
		Expect(rec.Header().Get("Location")).To(Equal("/v1/readableItems/" + createdID))
	})

	It("omits Location for a source that registers no item route", func() {
		rec := create("actionItems", "example")

		Expect(rec.Code).To(Equal(http.StatusAccepted))
		Expect(rec.Header()).ToNot(HaveKey("Location"))
	})

	It("omits Location for a source that suppresses it", func() {
		rec := create("secretItems", "example")

		Expect(rec.Code).To(Equal(http.StatusCreated))
		Expect(rec.Header()).ToNot(HaveKey("Location"))
	})
})
