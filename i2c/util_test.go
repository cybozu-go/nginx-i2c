package i2c

import (
	"reflect"
	"testing"
)

func Test_getSubnetsFromIPCount(t *testing.T) {
	type args struct {
		startIP string
		count   uint32
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "IPv4SingleSubnet",
			args: args{
				startIP: "192.0.2.0",
				count: 256,
			},
			want: []string{
				"192.0.2.0/24",
			},
		},
		{
			name: "IPv4MultipleSubnets",
			args: args{
				startIP: "192.0.2.0",
				count: 36,
			},
			want: []string{
				"192.0.2.0/27",
				"192.0.2.32/30",
			},
		},
		{
			name: "IPv6SingleSubnet",
			args: args{
				startIP: "2001:db8::",
				count: 4096,
			},
			want: []string{
				"2001:db8::/116",
			},
		},
		{
			name: "IPv4MultipleSubnets",
			args: args{
				startIP: "2001:db8::",
				count: 1536,
			},
			want: []string{
				"2001:db8::/118",
				"2001:db8::400/119",
			},
		},
		{
			name: "InvalidIP",
			args: args{
				startIP: "hoge",
				count: 36,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSubnetsFromIPCount(tt.args.startIP, tt.args.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSubnetsFromIPCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSubnetsFromIPCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
