create user pqgo;
create database pqgo;
alter role pqgo set experimental_enable_temp_tables=on;
alter role pqgo set autocommit_before_ddl=off;
alter role pqgo set default_int_size=4;
