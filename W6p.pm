#! /usr/bin/env perl
###################################################
#
#  Copyright (C) 2020 Sergey Voloshin <git@varrcan.me>
#
#  This file is part of Shutter.
#
#  Shutter is free software; you can redistribute it and/or modify
#  it under the terms of the GNU General Public License as published by
#  the Free Software Foundation; either version 3 of the License, or
#  (at your option) any later version.
#
#  Shutter is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
#
#  You should have received a copy of the GNU General Public License
#  along with Shutter; if not, write to the Free Software
#  Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA
#
###################################################

package W6p;

use lib $ENV{'SHUTTER_ROOT'} . '/share/shutter/resources/modules';

use utf8;
use strict;
use POSIX qw/setlocale/;
use Locale::gettext;
use Glib qw/TRUE FALSE/;
use Data::Dumper;

use Shutter::Upload::Shared;
our @ISA = qw(Shutter::Upload::Shared);

my $d = Locale::gettext->domain("shutter-upload-plugins");
$d->dir($ENV{'SHUTTER_INTL'});

my %upload_plugin_info = (
    'module'                     => "W6p",
    'url'                        => "https://w6p.ru/",
    'registration'               => "-",
    'name'                       => "W6p",
    'description'                => "Share image to w6p.ru",
    'supports_anonymous_upload'  => FALSE,
    'supports_authorized_upload' => FALSE,
    'supports_oauth_upload'      => TRUE,
);

binmode(STDOUT, ":utf8");
if (exists $upload_plugin_info{$ARGV[ 0 ]}) {
    print $upload_plugin_info{$ARGV[ 0 ]};
    exit;
}

###################################################

sub new {
    my $class = shift;

    #call constructor of super class (host, debug_cparam, shutter_root, gettext_object, main_gtk_window, ua)
    my $self = $class->SUPER::new(shift, shift, shift, shift, shift, shift);

    bless $self, $class;
    return $self;
}

sub init {
    my $self = shift;
    my $username = shift;

    use JSON::MaybeXS;
    use LWP::UserAgent;
    use HTTP::Request::Common;
    use Path::Class;

    $self->{_config} = {};
    $self->{_config_file} = file($ENV{'HOME'}, '/.shutter/shutter-config');

    $self->load_config;
    if (!$self->{_config}->{w6p_token}) {
        return $self->connect;
    }

    return TRUE;
}

sub load_config {
    my $self = shift;

    if (-f $self->{_config_file}) {
        eval {
            $self->{_config} = decode_json($self->{_config_file}->slurp);
        };
    }

    return TRUE;
}

sub connect {
    my $self = shift;
    return $self->setup;
}

sub setup {
    my $self = shift;

    my $sd = Shutter::App::SimpleDialogs->new;

    my $pin_entry = Gtk2::Entry->new();
    my $pin = '';
    $pin_entry->signal_connect(changed => sub {
        $pin = $pin_entry->get_text;
    });

    my $button = $sd->dlg_info_message(
        "Пожалуйста, введите токен из приложения WEBKPI", # message
        "Авторизация", # header
        'gtk-cancel', 'gtk-apply', undef, # button text
        undef, undef, undef, # button widget
        undef, # detail message
        undef, # detail checkbox
        $pin_entry, # content widget
        Gtk2::LinkButton->new ("https://cp.webpractik.ru/marketplace/app/8/", $d->get("WEBKPI")), # content widget2
    );

    if ($button == 20) {
        $self->{_config}->{w6p_token} = $pin;
        $self->{_config_file}->openw->print(encode_json($self->{_config}));
        chmod 0600, $self->{_config_file};

        return TRUE;
    }
    else {
        return FALSE;
    }
}

#handle
sub upload {
    my ($self, $upload_filename, $username, $password) = @_;

    #store as object vars
    $self->{_filename} = $upload_filename;
    $self->{_username} = $username;
    $self->{_password} = $password;

    utf8::encode $upload_filename;
    utf8::encode $password;
    utf8::encode $username;

    my $client = LWP::UserAgent->new(
        'timeout'    => 20,
        'keep_alive' => 10,
        'env_proxy'  => 1,
    );

    #upload the file
    eval {
        my $response = $client->request(POST 'https://w6p.ru/site/upload',
            Content_Type => 'form-data',
            Content      => [
                'token'                 => $self->{_config}->{w6p_token},
                'UploadForm[imageFile]' => [ $upload_filename ],
            ]
        );

        #TODO: добавить валидацию

        $self->{_links}->{'image'} = $response->content;
        $self->{_links}{'status'} = 200;

    };
    if ($@) {
        $self->{_links}{'status'} = $@;
    }

    #and return links
    return %{$self->{_links}};
}

1;
