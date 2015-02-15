#!/usr/bin/perl
use strict;
use warnings;

our $Url;
our $Name;
our $Tag;
our $mDate;
our $aDate;
our $Date;
our @Tag;
our $iTag;
our $currLine;

print "PRAGMA foreign_keys=OFF;\n";
print "BEGIN TRANSACTION;\n";
print "CREATE VIRTUAL TABLE tagmarks USING fts4(\n";
print "\t\"url\" TEXT NOT NULL,\n";
print "\t\"name\" TEXT NOT NULL,\n";
print "\t\"date\" TEXT NOT NULL,\n";
print "\t\"tags\" TEXT DEFAULT ('NULL')\n";
print ");\n";

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
			$Tag = $1;
			$Tag =~ s/TAGS=//gi;
			$Tag =~ s/"//gi;
		} else {
			$Tag = "NULL";
		}
		if($currLine =~ m/(LAST_MODIFIED="[^"]*")/) {
			$mDate = $1;
			$mDate =~ s/LAST_MODIFIED=//gi;
			$mDate =~ s/"//gi;
		} else {
			$mDate = time;
		}
		if($currLine =~ m/(ADD_DATE="[^"]*")/) {
			$aDate = $1;
			$aDate =~ s/ADD_DATE=//gi;
			$aDate =~ s/"//gi;
		} else {
			$aDate = time;
		}
		if ($mDate <= $aDate) {
			$Date = $aDate;
		} elsif ($mDate > $aDate) {
			$Date = $mDate;
		}
		@Tag = split(/,/, $Tag);
		@Tag = sort @Tag;
		$Tag = join(",", @Tag);
		print "INSERT INTO \"tagmarks\" VALUES('$Url','$Name','$Date','$Tag');\n";
	}
}
print "COMMIT;\n";