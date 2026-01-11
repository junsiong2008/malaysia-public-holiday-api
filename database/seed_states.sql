INSERT INTO states (code, name, name_ms, weekend_days, weekend_pattern, saturday_replacement_rule) VALUES
-- Fri-Sat States
('KDH', 'Kedah', 'Kedah', ARRAY['Friday', 'Saturday'], 'fri-sat', 'no_replacement'),
('KTN', 'Kelantan', 'Kelantan', ARRAY['Friday', 'Saturday'], 'fri-sat', 'replace_on_sunday'),
('TRG', 'Terengganu', 'Terengganu', ARRAY['Friday', 'Saturday'], 'fri-sat', 'replace_on_sunday'),

-- Sat-Sun States
('JHR', 'Johor', 'Johor', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('KUL', 'Kuala Lumpur', 'Wilayah Persekutuan Kuala Lumpur', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('LBN', 'Labuan', 'Wilayah Persekutuan Labuan', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('MLK', 'Melaka', 'Melaka', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('NSN', 'Negeri Sembilan', 'Negeri Sembilan', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('PHG', 'Pahang', 'Pahang', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('PNG', 'Penang', 'Pulau Pinang', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('PRK', 'Perak', 'Perak', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('PLS', 'Perlis', 'Perlis', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('SBH', 'Sabah', 'Sabah', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('SGR', 'Selangor', 'Selangor', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('SWK', 'Sarawak', 'Sarawak', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement'),
('WP', 'Putrajaya', 'Wilayah Persekutuan Putrajaya', ARRAY['Saturday', 'Sunday'], 'sat-sun', 'no_replacement');
