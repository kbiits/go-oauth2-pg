package pg

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	pgAdapter "github.com/vgarvardt/go-pg-adapter"
)

func TestClientStore_initTable(t *testing.T) {
	adapter := new(mockAdapter)

	adapter.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		query := args.Get(1).(string)
		// new line character is the character at position 0
		assert.Equal(t, 1, strings.Index(query, "CREATE TABLE IF NOT EXISTS"))
	})

	_, err := NewClientStore(adapter)
	require.NoError(t, err)

	adapter.AssertExpectations(t)
}

func TestClientStore_GetByID(t *testing.T) {
	type fields struct {
		adapter           pgAdapter.Adapter
		tableName         string
		logger            Logger
		initTableDisabled bool
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		mock    func(*mockAdapter)
		args    args
		want    oauth2.ClientInfo
		wantErr error
	}{
		{
			name: "Success scenario - id is empty",
			fields: fields{
				tableName:         "a",
				logger:            &memoryLogger{},
				initTableDisabled: true,
			},
			args: args{
				id:  "",
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: nil,
			mock:    func(ma *mockAdapter) {},
		},
		{
			name: "Success scenario - success get client data",
			fields: fields{
				tableName:         "a",
				logger:            &memoryLogger{},
				initTableDisabled: true,
			},
			args: args{
				id:  "abc",
				ctx: context.Background(),
			},
			want: &models.Client{
				ID:     "abc",
				Secret: "",
				Domain: "",
				Public: false,
				UserID: "abc",
			},
			wantErr: nil,
			mock: func(m *mockAdapter) {
				mock := m.On("SelectOne", []interface{}{context.Background(), mock.MatchedBy(func(value interface{}) bool {
					item, ok := value.(*ClientStoreItem)
					if ok {
						item.Data = []byte(`
							{
								"ID": "abc",
								"Secret": "",
								"Domain": "",
								"Public": false,
								"UserID": "abc"
							}
						`)
					}

					return ok
				}), "SELECT id, secret, domain, data FROM a WHERE id = $1", []interface{}{"abc"}}...)

				mock.Return(nil)
			},
		},
		{
			name: "Failed scenario - error in sql adapter level",
			fields: fields{
				tableName:         "a",
				logger:            &memoryLogger{},
				initTableDisabled: true,
			},
			args: args{
				id:  "abc",
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: errors.New("something went wrong"),
			mock: func(m *mockAdapter) {
				m.On("SelectOne", []interface{}{context.Background(), mock.Anything, "SELECT id, secret, domain, data FROM a WHERE id = $1", []interface{}{"abc"}}...).
					Return(errors.New("something went wrong"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdptr := &mockAdapter{}
			tt.mock(mockAdptr)

			tt.fields.adapter = mockAdptr
			s := &ClientStore{
				adapter:           tt.fields.adapter,
				tableName:         tt.fields.tableName,
				logger:            tt.fields.logger,
				initTableDisabled: tt.fields.initTableDisabled,
			}

			got, err := s.GetByID(tt.args.ctx, tt.args.id)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
				return
			}

			require.EqualValues(t, tt.want, got)
		})
	}
}
