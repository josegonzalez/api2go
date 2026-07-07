package api2go

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// nestedItem is a minimal resource entity used to exercise NestedResource
// routing.
type nestedItem struct {
	ID   string `json:"-"`
	Name string `json:"name"`
}

func (n nestedItem) GetID() string          { return n.ID }
func (n *nestedItem) SetID(id string) error { n.ID = id; return nil }
func (n nestedItem) GetName() string        { return "nestedItems" }

// nestedItemSource records what its handlers observed so the specs can assert
// that the nested routes inject the parent id and route the child id through to
// the flat handlers.
type nestedItemSource struct {
	createParentID string
	deletedID      string
	deleteParentID string
}

func (s *nestedItemSource) NestConfig() NestConfig {
	return NestConfig{Parent: "parents", Relation: "nestedItems", ChildParam: "itemId"}
}

func nestedParentID(req Request, key string) string {
	if v, ok := req.QueryParams[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func (s *nestedItemSource) FindAll(req Request) (Responder, error) {
	return &Response{Res: []nestedItem{}}, nil
}

func (s *nestedItemSource) FindOne(id string, req Request) (Responder, error) {
	return &Response{Res: nestedItem{ID: id}}, nil
}

func (s *nestedItemSource) Create(obj interface{}, req Request) (Responder, error) {
	s.createParentID = nestedParentID(req, "parentsID")
	item := obj.(nestedItem)
	item.ID = "generated"
	return &Response{Res: item, Code: http.StatusCreated}, nil
}

func (s *nestedItemSource) Delete(id string, req Request) (Responder, error) {
	s.deletedID = id
	s.deleteParentID = nestedParentID(req, "parentsID")
	return &Response{Code: http.StatusNoContent}, nil
}

var _ = Describe("NestedResource routing", func() {
	var (
		api    *API
		source *nestedItemSource
	)

	BeforeEach(func() {
		source = &nestedItemSource{}
		api = NewAPIWithRouting(testPrefix, NewStaticResolver(""), newTestRouter())
		api.AddResource(nestedItem{}, source)
	})

	It("creates through the nested route and injects the parent id", func() {
		body := strings.NewReader(`{"data": {"type": "nestedItems", "attributes": {"name": "example"}}}`)
		rec := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/v1/parents/my-parent/nestedItems", body)
		Expect(err).To(BeNil())
		api.Handler().ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusCreated))
		Expect(source.createParentID).To(Equal("my-parent"))
	})

	It("deletes through the nested route with the child id and parent id", func() {
		rec := httptest.NewRecorder()
		req, err := http.NewRequest("DELETE", "/v1/parents/my-parent/nestedItems/child-1", nil)
		Expect(err).To(BeNil())
		api.Handler().ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusNoContent))
		Expect(source.deletedID).To(Equal("child-1"))
		Expect(source.deleteParentID).To(Equal("my-parent"))
	})

	It("keeps the flat routes working without a parent id", func() {
		body := strings.NewReader(`{"data": {"type": "nestedItems", "attributes": {"name": "example"}}}`)
		rec := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/v1/nestedItems", body)
		Expect(err).To(BeNil())
		api.Handler().ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusCreated))
		Expect(source.createParentID).To(Equal(""))
	})
})
