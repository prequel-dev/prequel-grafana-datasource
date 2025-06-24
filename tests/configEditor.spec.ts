import { test, expect } from '@grafana/plugin-e2e';
import { PreqDataSourceOptions, PreqSecureJsonData } from '../src/types';

test('smoke: should render config editor', async ({ createDataSourceConfigPage, readProvisionedDataSource, page }) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await createDataSourceConfigPage({ type: ds.type });
  await expect(page.getByLabel('Token')).toBeVisible();
});
test('"Save & test" should be successful when configuration is valid', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource<PreqDataSourceOptions, PreqSecureJsonData>({ fileName: 'datasources.yml' });
  const configPage = await createDataSourceConfigPage({ type: ds.type });
  await page.getByRole('textbox', { name: 'URL' }).fill(ds.jsonData.url ?? '');
  await page.getByRole('textbox', { name: 'Token' }).fill(ds.secureJsonData?.token ?? '');
  await expect(configPage.saveAndTest()).toBeOK();
});

test('"Save & test" should fail when configuration is invalid', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource<PreqDataSourceOptions, PreqSecureJsonData>({ fileName: 'datasources.yml' });
  const configPage = await createDataSourceConfigPage({ type: ds.type });
  await page.getByRole('textbox', { name: 'URL' }).fill(ds.jsonData.url ?? '');
  await expect(configPage.saveAndTest()).not.toBeOK();
  await expect(configPage).toHaveAlert('error', { hasText: 'Token is missing' });
});
