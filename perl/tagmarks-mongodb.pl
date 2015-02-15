#!/usr/bin/perl
#cat bookmarks.html | ./tagmarks-mongodb.pl > tagmarks.js
use strict;
use warnings;

our $Url;
our $Name;
our $Tags;
our @Tags;
our $iTag;
our $currLine;

print "db.tagmarks.drop()\n";
print "db.createCollection('tagmarks',{autoIndexId:false})\n";
while (<STDIN>) {
	$currLine = $_;
	chomp $currLine;
	if($currLine =~ m/(<A HREF="[^"]*")/) {
		$Url = $1;
		$Url =~ s/<A HREF=//gi;
		$Url =~ s/"//gi;
		if($currLine =~ m/(>[^"]*<\/A>)/) {
			$Name = $1;
			$Name =~ s/<\/A>//gi;
			$Name =~ s/>//gi;
		} else {
			$Name = "NULL";
		}
		if($currLine =~ m/(TAGS="[^"]*")/) {
			$Tags = $1;
			$Tags =~ s/TAGS=//gi;
			$Tags =~ s/"//gi;
		} else {
			$Tags = "NULL";
		}
		@Tags = split(/,/, $Tags);
		@Tags = sort @Tags;
		$Tags = join("','", @Tags);
		print "db.tagmarks.insert({url:'$Url',name:'$Name',tags:['$Tags']})\n";
	}
}
